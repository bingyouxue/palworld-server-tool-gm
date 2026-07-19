# SPDX-License-Identifier: Apache-2.0
# Derived from zaigie/palworld-server-tool sav_cli @ fb45624 (Apache-2.0).
# Runtime deps (palsav-flex/palooz/ooz) are GPL-3.0-or-later, so a Docker image
# built from the root Dockerfile includes these runtime components.
"""Decode a Palworld 1.0 save and structure it into player / guild JSON.

It uses the ``palsav`` parser from PalworldSaveTools, which ships Palworld 1.0
mappings (GroupSaveDataMap / character / item-container decoders) plus Oodle
(``PlM1``) decompression via the native ``palooz`` module.

The parser performs a full decode. A ~260KB compressed / ~4MB decompressed
Level.sav completes in a couple of seconds on the validated fixtures.
"""

import os

from palsav.core import decompress_sav_to_gvas
from palsav.gvas import GvasFile
from palsav.paltypes import PALWORLD_TYPE_HINTS, PALWORLD_CUSTOM_PROPERTIES

from world_types import Player, Pal, Guild, BaseCamp
from logger import log

# Global state shared by the current decode helpers.
wsd = None
gvas_file = None

PLAYER_CONTAINER_KEYS = [
    "CommonContainerId",
    "DropSlotContainerId",
    "EssentialContainerId",
    "FoodEquipContainerId",
    "PlayerEquipArmorContainerId",
    "WeaponLoadOutContainerId",
]


def _read_gvas(path):
    with open(path, "rb") as f:
        raw_gvas, _ = decompress_sav_to_gvas(f.read())
    return GvasFile.read(raw_gvas, PALWORLD_TYPE_HINTS, PALWORLD_CUSTOM_PROPERTIES)


def convert_sav(file):
    """Decode Level.sav into the module-global ``wsd`` (worldSaveData)."""
    global gvas_file, wsd
    gvas_file = _read_gvas(file)
    wsd = gvas_file.properties["worldSaveData"]["value"]
    return wsd


def _save_parameter(character_entry):
    return character_entry["value"]["RawData"]["value"]["object"]["SaveParameter"][
        "value"
    ]


def _index_character_containers():
    """Map instance_id (str) -> (container_id str, slot_index int).

    CharacterContainerSaveData stores each slot as a decoded RawData blob with
    fields: player_uid, instance_id, permission_tribe_id.
    The container key is the container ID (UUID).  We build a lookup from
    pal instance_id -> (container_id, slot_index) so we can later tell whether
    a pal lives in the player's party (OtomoCharacterContainerId) or PalBox
    (PalStorageContainerId).
    """
    index = {}  # instance_id_str -> (container_id_str, slot_index)
    if not wsd.get("CharacterContainerSaveData"):
        return index
    for entry in wsd["CharacterContainerSaveData"]["value"]:
        container_id = str(entry["key"]["ID"]["value"])
        slots = entry["value"].get("Slots", {}).get("value", {}).get("values", [])
        for slot_idx, slot in enumerate(slots):
            raw = slot.get("RawData", {}).get("value")
            if not raw:
                continue
            instance_id = str(raw.get("instance_id", ""))
            if instance_id:
                index[instance_id] = (container_id, slot_idx)
    return index


def _index_player_pal_containers(dir_path):
    """Map player_uid_decimal -> {"otomo": container_id, "storage": container_id}.

    In Palworld 1.0, OtomoCharacterContainerId and PalStorageContainerId are
    top-level fields of SaveData, NOT nested inside InventoryInfo.
      OtomoCharacterContainerId -> party / backpack (max 5 pals)
      PalStorageContainerId     -> PalBox (箱内)
    """
    result = {}
    if not dir_path or not os.path.exists(dir_path):
        return result

    def _cid(ref):
        try:
            return str(ref["value"]["ID"]["value"])
        except (KeyError, TypeError):
            return ""

    for fname in os.listdir(dir_path):
        if not fname.upper().endswith(".SAV"):
            continue
        fpath = os.path.join(dir_path, fname)
        try:
            player_gvas = _read_gvas(fpath).properties["SaveData"]["value"]
        except Exception as _e:
            log(
                f"Could not read player containers from {fname}: "
                f"{type(_e).__name__}: {_e}",
                "WARNING",
            )
            continue

        # OtomoCharacterContainerId and PalStorageContainerId are top-level
        otomo_id   = _cid(player_gvas.get("OtomoCharacterContainerId"))
        storage_id = _cid(player_gvas.get("PalStorageContainerId"))
        if not otomo_id and not storage_id:
            continue

        uid_hex = fname.upper().replace(".SAV", "")
        try:
            uid_decimal = str(int(uid_hex[:8], 16))
        except (ValueError, IndexError):
            uid_decimal = uid_hex

        result[uid_decimal] = {"otomo": otomo_id, "storage": storage_id}
    return result


def _is_in_palbox(instance_id, char_containers, player_pal_containers):
    """Return True unless the pal is confirmed to be in the player's otomo (party) container.

    Classification:
      - otomo container (OtomoCharacterContainerId)  → False  (party / backpack, max 5)
      - everything else: PalStorage, base-camp workers, unmatched → True  (palbox tab)

    Base-camp workers and pals in the PalTerminal are NOT in any CharacterContainer
    slot at all, so char_containers.get() returns None for them — they fall through
    to the default True, which is correct (they belong on the "palbox" tab).
    """
    if not instance_id:
        return True
    entry = char_containers.get(instance_id)
    if entry is None:
        # Not in any character container (base-camp worker, pal terminal, etc.)
        return True
    container_id, _slot = entry
    # Only return False (= party/backpack) when we positively identify the otomo container
    for containers in player_pal_containers.values():
        if container_id == containers.get("otomo"):
            return False
    # In a container that is not otomo: PalStorage or unknown → treat as palbox
    return True


def _index_guild_members():
    """Return a dict mapping guild_id -> list[player_uid str] for all guilds.

    Used to attribute base-camp (no-owner) pals to a guild member when the
    guild has only one member, or to the first available member otherwise.
    """
    if not wsd.get("GroupSaveDataMap"):
        return {}
    guild_members = {}
    for g in wsd["GroupSaveDataMap"]["value"]:
        if g["value"]["GroupType"]["value"]["value"] != "EPalGroupType::Guild":
            continue
        # g["key"] is a UUID object, not a dict — convert directly
        gid   = str(g["key"])
        raw_g = g["value"]["RawData"]["value"]
        members = [str(m["player_uid"]) for m in raw_g.get("players", [])]
        if members:
            guild_members[gid] = members
    return guild_members


def structure_player(dir_path, filetime: int = -1):
    if not wsd.get("CharacterSaveParameterMap"):
        return [], 0

    ticks = wsd["GameTimeSaveData"]["value"]["RealDateTimeTicks"]["value"]
    item_containers = _index_item_containers()
    char_containers = _index_character_containers()
    player_pal_containers = _index_player_pal_containers(dir_path)
    guild_members = _index_guild_members()
    # Flat list of all guild member uids (for no-owner pal attribution)
    all_guild_uids = [uid for uids in guild_members.values() for uid in uids]

    NULL_UUID = "00000000-0000-0000-0000-000000000000"

    players = []
    owned_pals = []    # pals with a real OwnerPlayerUId
    noowner_pals = []  # base-camp / shared pals with null owner
    player_save_warnings = 0
    for c in wsd["CharacterSaveParameterMap"]["value"]:
        uid = c["key"]["PlayerUId"]["value"]
        instance_id = str(c["key"].get("InstanceId", {}).get("value", ""))
        sp = _save_parameter(c)
        if sp.get("IsPlayer") and sp["IsPlayer"]["value"]:
            sp["Items"], has_warning = getPlayerItems(
                uid, dir_path, item_containers
            )
            player_save_warnings += int(has_warning)
            players.append(Player(uid, sp).to_dict())
        else:
            owner_uid = sp.get("OwnerPlayerUId", {}).get("value", None)
            owner_str = str(owner_uid) if owner_uid else ""
            in_palbox = _is_in_palbox(
                instance_id, char_containers, player_pal_containers
            )
            pal_dict = Pal(sp, ticks, filetime, in_palbox=in_palbox).to_dict()
            if owner_str and owner_str != NULL_UUID:
                owned_pals.append(pal_dict)
            else:
                pal_dict["is_base_pal"] = True
                # Base-camp / shared pal: no owner — will be attributed to a
                # guild member after players are collected.
                noowner_pals.append(pal_dict)

    # De-dup players by uid, keeping the highest-level record.
    unique_players_dict = {}
    for player in players:
        pid = player["player_uid"]
        if pid not in unique_players_dict or player["level"] > unique_players_dict[pid]["level"]:
            unique_players_dict[pid] = player
    unique_players = list(unique_players_dict.values())

    # Attach owned pals to their player
    for pal in owned_pals:
        for player in unique_players:
            if player["player_uid"] == pal["owner"]:
                pal.pop("owner")
                player["pals"].append(pal)
                break

    # Attribute no-owner (base-camp) pals to the first matching guild member
    # that is present in unique_players.  If there is only one player, all
    # base-camp pals go to that player.
    if noowner_pals:
        player_uid_set = {p["player_uid"] for p in unique_players}
        # Prefer guild members that are actually in this save
        target_uid = None
        for uid in all_guild_uids:
            if uid in player_uid_set:
                target_uid = uid
                break
        if target_uid is None and unique_players:
            # Fallback: first player in the save
            target_uid = unique_players[0]["player_uid"]
        if target_uid:
            for player in unique_players:
                if player["player_uid"] == target_uid:
                    for pal in noowner_pals:
                        pal.pop("owner", None)
                        player["pals"].append(pal)
                    break

    return (
        sorted(unique_players, key=lambda p: p["level"], reverse=True),
        player_save_warnings,
    )


def _index_item_containers():
    """Map container-UUID string -> decoded slots list."""
    index = {}
    if not wsd.get("ItemContainerSaveData"):
        return index
    for container in wsd["ItemContainerSaveData"]["value"]:
        cid = str(container["key"]["ID"]["value"])
        index[cid] = container["value"]["Slots"]["value"]["values"]
    return index


def getPlayerItems(player_uid, dir_path, item_containers):
    containers_data = {k: [] for k in PLAYER_CONTAINER_KEYS}

    player_sav_file = os.path.join(
        dir_path, str(player_uid).upper().replace("-", "") + ".sav"
    )
    if not os.path.exists(player_sav_file):
        return containers_data, True

    try:
        player_gvas = _read_gvas(player_sav_file).properties["SaveData"]["value"]
    except Exception as e:
        log(
            f"Skipped corrupted player save: {os.path.basename(player_sav_file)}: "
            f"{type(e).__name__}: {e}",
            "WARNING",
        )
        return containers_data, True

    inv = player_gvas.get("InventoryInfo")
    if inv is None:
        return containers_data, False

    for key in PLAYER_CONTAINER_KEYS:
        ref = inv["value"].get(key)
        if ref is None:
            continue
        container_id = str(ref["value"]["ID"]["value"])
        slots = item_containers.get(container_id)
        if slots is None:
            continue
        items = []
        for slot in slots:
            raw = slot["RawData"]["value"]
            if not raw:  # empty slot decodes to None
                continue
            static_id = raw["item"]["static_id"]
            if not static_id or static_id.lower() == "none":
                continue
            items.append(
                {
                    "SlotIndex": raw["slot_index"],
                    "ItemId": static_id.lower(),
                    "StackCount": raw["count"],
                }
            )
        containers_data[key] = items
    return containers_data, False


def structure_base_camp():
    if not wsd.get("BaseCampSaveData"):
        return []
    return [
        BaseCamp(b["value"]["RawData"]["value"]).to_dict()
        for b in wsd["BaseCampSaveData"]["value"]
    ]


def structure_guild(filetime: int = -1):
    if not wsd.get("GroupSaveDataMap"):
        return []
    base_camps = structure_base_camp()
    ticks = wsd["GameTimeSaveData"]["value"]["RealDateTimeTicks"]["value"]
    groups = (
        g["value"]["RawData"]["value"]
        for g in wsd["GroupSaveDataMap"]["value"]
        if g["value"]["GroupType"]["value"]["value"] == "EPalGroupType::Guild"
    )
    sorted_guilds = sorted(
        (Guild(g, ticks, filetime).to_dict() for g in groups),
        key=lambda g: g["base_camp_level"],
        reverse=True,
    )
    for guild in sorted_guilds:
        for camp in base_camps:
            if camp["id"] in guild["base_ids"]:
                guild["base_camp"].append(
                    {
                        "id": camp["id"],
                        "area": camp["area_range"],
                        "location_x": camp["transform"]["x"],
                        "location_y": camp["transform"]["y"],
                    }
                )
    return list(sorted_guilds)
