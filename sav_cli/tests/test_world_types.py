import sys
import unittest
from pathlib import Path


sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from world_types import Pal

# Minimal SaveParameter dict with only the required OwnerPlayerUId key.
def _make_pal(**kwargs):
    data = {"OwnerPlayerUId": {"value": "00000001-0000-0000-0000-000000000000"}}
    data.update(kwargs)
    return Pal(data, real_date_time_ticks=0, filetime=0).to_dict()


class PalOutputTests(unittest.TestCase):

    # ── IV / talent fields ────────────────────────────────────────────────────

    def test_talent_hp_exposed_as_talent_hp_and_legacy_melee(self):
        pal = _make_pal(Talent_HP={"value": {"value": 73}})
        self.assertEqual(pal["talent_hp"], 73)
        # Legacy alias must remain for backward compatibility
        self.assertEqual(pal["melee"], 73)

    def test_talent_shot_exposed_as_talent_shot_and_legacy_ranged(self):
        pal = _make_pal(Talent_Shot={"value": {"value": 55}})
        self.assertEqual(pal["talent_shot"], 55)
        self.assertEqual(pal["ranged"], 55)

    def test_talent_defense_exposed_as_talent_defense_and_legacy_defense(self):
        pal = _make_pal(Talent_Defense={"value": {"value": 40}})
        self.assertEqual(pal["talent_defense"], 40)
        self.assertEqual(pal["defense"], 40)

    def test_missing_iv_fields_default_to_zero(self):
        pal = _make_pal()
        self.assertEqual(pal["talent_hp"], 0)
        self.assertEqual(pal["talent_shot"], 0)
        self.assertEqual(pal["talent_defense"], 0)

    # ── Passive skills ────────────────────────────────────────────────────────

    def test_passive_skills_from_PassiveSkillList(self):
        pal = _make_pal(
            PassiveSkillList={"value": {"values": ["Legend", "Swift"]}}
        )
        self.assertEqual(pal["passive_skills"], ["Legend", "Swift"])
        # Legacy alias
        self.assertEqual(pal["skills"], ["Legend", "Swift"])

    def test_missing_passive_skills_default_to_empty_list(self):
        pal = _make_pal()
        self.assertEqual(pal["passive_skills"], [])
        self.assertEqual(pal["skills"], [])

    # ── Active skills ─────────────────────────────────────────────────────────

    def test_active_skills_from_EquipWaza_name_strings(self):
        pal = _make_pal(
            EquipWaza={"value": {"values": ["EPalWazaID::AirCanon", "EPalWazaID::Bite"]}}
        )
        self.assertEqual(pal["active_skills"], ["EPalWazaID::AirCanon", "EPalWazaID::Bite"])

    def test_active_skills_from_EquipWaza_dict_values(self):
        # Some palsav versions wrap each entry as {"value": "..."}
        pal = _make_pal(
            EquipWaza={"value": {"values": [
                {"value": "EPalWazaID::AirCanon"},
                {"value": "EPalWazaID::Bite"},
            ]}}
        )
        self.assertEqual(pal["active_skills"], ["EPalWazaID::AirCanon", "EPalWazaID::Bite"])

    def test_missing_equip_waza_defaults_to_empty_list(self):
        pal = _make_pal()
        self.assertEqual(pal["active_skills"], [])

    def test_mastered_skills_from_MasteredWaza(self):
        pal = _make_pal(
            MasteredWaza={"value": {"values": ["EPalWazaID::AirCanon", "EPalWazaID::Bite", "EPalWazaID::DragonCannon"]}}
        )
        self.assertEqual(len(pal["mastered_skills"]), 3)

    # ── Stars / Rank ──────────────────────────────────────────────────────────

    def test_stars_is_rank_minus_one(self):
        pal = _make_pal(Rank={"value": {"value": 4}})
        self.assertEqual(pal["rank"], 4)
        self.assertEqual(pal["stars"], 3)

    def test_stars_clamp_to_zero_for_rank_one(self):
        # Rank=1 means uncondensed; stars must be 0, not negative
        pal = _make_pal(Rank={"value": {"value": 1}})
        self.assertEqual(pal["stars"], 0)

    def test_missing_rank_defaults_to_rank_1_stars_0(self):
        pal = _make_pal()
        self.assertEqual(pal["rank"], 1)
        self.assertEqual(pal["stars"], 0)

    # ── in_palbox location flag ───────────────────────────────────────────────

    def test_in_palbox_defaults_to_false(self):
        pal = _make_pal()
        self.assertFalse(pal["in_palbox"])

    def test_in_palbox_true_when_passed(self):
        data = {"OwnerPlayerUId": {"value": "00000001-0000-0000-0000-000000000000"}}
        pal = Pal(data, real_date_time_ticks=0, filetime=0, in_palbox=True).to_dict()
        self.assertTrue(pal["in_palbox"])

    # ── Gender / type / lucky ─────────────────────────────────────────────────

    def test_gender_parsed_from_enum_value(self):
        pal = _make_pal(
            Gender={"value": {"value": "EPalGenderType::Male"}}
        )
        self.assertEqual(pal["gender"], "Male")

    def test_is_lucky_from_IsRarePal(self):
        pal = _make_pal(IsRarePal={"value": True})
        self.assertTrue(pal["is_lucky"])

    def test_character_id_parsed_as_type(self):
        pal = _make_pal(CharacterID={"value": "Penguin"})
        self.assertEqual(pal["type"], "Penguin")

    def test_boss_prefix_sets_is_boss(self):
        pal = _make_pal(CharacterID={"value": "BOSS_Penguin"})
        self.assertTrue(pal["is_boss"])

    # ── Output dict completeness ──────────────────────────────────────────────

    def test_all_expected_keys_present(self):
        pal = _make_pal()
        expected = {
            "nickname", "level", "exp", "hp", "max_hp", "type", "gender",
            "is_lucky", "is_boss", "is_tower", "workspeed",
            "talent_hp", "talent_shot", "talent_defense",
            "melee", "ranged", "defense",
            "rank", "rank_attack", "rank_defence", "rank_craftspeed", "stars",
            "passive_skills", "active_skills", "mastered_skills", "skills",
            "in_palbox",
        }
        missing = expected - set(pal.keys())
        self.assertFalse(missing, f"Missing keys in Pal.to_dict(): {missing}")


if __name__ == "__main__":
    unittest.main()
