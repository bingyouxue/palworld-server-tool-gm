/*
 * PST Bridge DLL - direct ProcessEvent call with SEH protection
 * Offsets (Dumper-7, 5.1.1-0+++UE5+Release-5.1-Pal):
 *   ProcessEvent abs = 0x034EE780
 *   AppendString abs  = 0x03370650
 *   GObjects          = 0x08CCDE80
 *   GWorld            = 0x08E2A628
 */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <string>
#include <mutex>
#include <cstdint>
#include <cstring>

using int8   = signed char;
using int16  = short;
using int32  = int;
using int64  = long long;
using uint8  = unsigned char;
using uint16 = unsigned short;
using uint32 = unsigned int;
using uint64 = unsigned long long;

static const uintptr_t OFF_PROCESS_EVENT = 0x034EE780;
static const uintptr_t OFF_APPENDSTRING  = 0x03370650;
static const uintptr_t OFF_GOBJECTS      = 0x08CCDE80;
static const uintptr_t OFF_GWORLD        = 0x08E2A628;

static uintptr_t Base(){
    static uintptr_t b=0;
    if(!b) b=(uintptr_t)GetModuleHandleW(nullptr);
    return b;
}

struct FString {
    wchar_t* Data=nullptr; int32 Count=0; int32 Max=0;
    FString()=default;
    explicit FString(const wchar_t* s){
        if(!s)return;
        Count=(int32)wcslen(s)+1; Max=Count;
        Data=new wchar_t[Count]; memcpy(Data,s,Count*2);
    }
    ~FString(){delete[] Data;}
    FString(const FString&)=delete;
    FString& operator=(const FString&)=delete;
    std::string ToStd()const{
        if(!Data||Count<=1)return{};
        return std::string(Data,Data+Count-1);
    }
};
struct FName    {int32 ComparisonIndex=0;int32 Number=0;};
struct FGuid    {uint32 A=0,B=0,C=0,D=0;bool IsZero()const{return!A&&!B&&!C&&!D;}};
struct FText    {uint8 Pad[0x18]{};};
struct UObject  {void** VTable=nullptr;uint32 Flags=0;int32 Idx=0;void* Class=nullptr;FName Name{};void* Outer=nullptr;};
struct UFunction:UObject{uint32 FuncFlags=0;uint8 Pad[0x28]{};};

struct FUObjectItem{UObject* Obj=nullptr;int32 Flags=0,Cluster=0,Serial=0;};
struct TUObjectArray{
    FUObjectItem** Chunks=nullptr;uint8* Pre=nullptr;
    int32 MaxElems=0,NumElems=0,MaxChunks=0,NumChunks=0;
    static const int32 CHUNK=65536;
    UObject* Get(int32 i)const{
        if(i<0||i>=NumElems)return nullptr;
        int32 c=i/CHUNK,o=i%CHUNK;
        if(c>=NumChunks||!Chunks[c])return nullptr;
        return Chunks[c][o].Obj;
    }
    int32 Num()const{return NumElems;}
};
static TUObjectArray* GObjects(){return(TUObjectArray*)(Base()+OFF_GOBJECTS);}
static UObject*       GWorld()  {return*(UObject**)(Base()+OFF_GWORLD);}

// ── All UE calls go through SafeCall (SEH) ────────────────────────────────────
using AppendStringFn  = void(*)(const FName*,FString*);
using ProcessEventFn  = void(*)(UObject*,UFunction*,void*);

static AppendStringFn g_AppendString = nullptr;
static ProcessEventFn g_ProcessEvent = nullptr;

static int seh_appendstring(const FName* n, FString* s) {
    __try { g_AppendString(n,s); return 1; }
    __except(EXCEPTION_EXECUTE_HANDLER){ return 0; }
}
static int seh_processevent(UObject* obj, UFunction* func, void* parms) {
    __try { g_ProcessEvent(obj,func,parms); return 1; }
    __except(EXCEPTION_EXECUTE_HANDLER){ return 0; }
}

static std::string FNameToStr(FName n){
    FString s;
    if(!seh_appendstring(&n,&s)) return {};
    return s.ToStd();
}
static bool NameIs(UObject* o,const char* s){return o&&FNameToStr(o->Name)==s;}

static FName MakeFName(const std::string& target){
    auto* g=GObjects(); if(!g) return {};
    int32 n=g->Num();
    for(int32 i=0;i<n;i++){
        UObject* o=g->Get(i); if(!o||!o->Name.ComparisonIndex) continue;
        if(FNameToStr(o->Name)==target){FName r{};r.ComparisonIndex=o->Name.ComparisonIndex;return r;}
    }
    return {};
}
static UFunction* FindUFunc(const char* cls,const char* fn){
    auto* g=GObjects(); int32 n=g->Num();
    for(int32 i=0;i<n;i++){
        UObject* o=g->Get(i); if(!o||!o->Class) continue;
        if(!NameIs((UObject*)o->Class,"Function")) continue;
        if(!NameIs(o,fn)) continue;
        if(o->Outer&&!NameIs((UObject*)o->Outer,cls)) continue;
        return (UFunction*)o;
    }
    return nullptr;
}
static UObject* GetCDO(const char* cls){
    std::string name=std::string("Default__")+cls;
    auto* g=GObjects(); int32 n=g->Num();
    for(int32 i=0;i<n;i++){
        UObject* o=g->Get(i); if(!o) continue;
        if(FNameToStr(o->Name)==name) return o;
    }
    return nullptr;
}
static UObject* FindCompOnObj(UObject* owner,const char* compClass){
    auto* g=GObjects(); int32 n=g->Num();
    for(int32 i=0;i<n;i++){
        UObject* o=g->Get(i); if(!o||!o->Class) continue;
        if((UObject*)o->Outer!=owner) continue;
        if(NameIs((UObject*)o->Class,compClass)) return o;
    }
    return nullptr;
}
static void PE(UObject* self,UFunction* func,void* parms){
    seh_processevent(self,func,parms);
}

struct P_GetUID  {UObject* World=nullptr;FString UID{};FGuid Ret{};};
struct P_GetChar {UObject* World=nullptr;FGuid   UID{};UObject* Ret=nullptr;};
struct P_AddItem {FName Item{};int32 Count=0;bool Passive=false;};
struct P_Kick    {FString UID{};FText Reason{};};

static UFunction *FN_GetUID=nullptr,*FN_GetChar=nullptr,
                 *FN_AddItem=nullptr,*FN_Kick=nullptr;
static std::mutex gFuncMu;
static bool EnsureFuncs(){
    std::lock_guard<std::mutex> lk(gFuncMu);
    if(!FN_GetUID)  FN_GetUID  =FindUFunc("PalUtility","GetPlayerUIdByString");
    if(!FN_GetChar) FN_GetChar =FindUFunc("PalUtility","GetPlayerCharacterByPlayerUID");
    if(!FN_AddItem) FN_AddItem =FindUFunc("PalPlayerInventoryData","RequestAddItem_ForDebug");
    if(!FN_Kick)    FN_Kick    =FindUFunc("PalCheatManager","KickPlayer");
    return FN_GetUID&&FN_GetChar&&FN_AddItem&&FN_Kick;
}

static std::string DoGiveItem(const std::string& uid,const std::string& item,int cnt){
    if(!EnsureFuncs()) return "UFunctions not ready";
    UObject* world=GWorld(); if(!world) return "GWorld null";
    UObject* cdo=GetCDO("PalUtility"); if(!cdo) return "PalUtility CDO not found";

    uint32 f1=FN_GetUID->FuncFlags; FN_GetUID->FuncFlags|=0x400;
    P_GetUID p1; p1.World=world;
    new(&p1.UID) FString(std::wstring(uid.begin(),uid.end()).c_str());
    PE(cdo,FN_GetUID,&p1);
    FN_GetUID->FuncFlags=f1;
    if(p1.Ret.IsZero()) return "player not found: "+uid;

    uint32 f2=FN_GetChar->FuncFlags; FN_GetChar->FuncFlags|=0x400;
    P_GetChar p2; p2.World=world; p2.UID=p1.Ret;
    PE(cdo,FN_GetChar,&p2);
    FN_GetChar->FuncFlags=f2;
    if(!p2.Ret) return "player character not found";

    UObject* inv=FindCompOnObj(p2.Ret,"PalPlayerInventoryData");
    if(!inv) return "PlayerInventoryData not found";

    FName itemName=MakeFName(item);
    if(!itemName.ComparisonIndex) return "item not found: "+item;

    uint32 f3=FN_AddItem->FuncFlags; FN_AddItem->FuncFlags|=0x400;
    P_AddItem p3; p3.Item=itemName; p3.Count=cnt; p3.Passive=false;
    PE(inv,FN_AddItem,&p3);
    FN_AddItem->FuncFlags=f3;
    return "ok";
}
static std::string DoKick(const std::string& uid){
    if(!EnsureFuncs()) return "UFunctions not ready";
    UObject* cm=nullptr;
    auto* g=GObjects(); int32 n=g->Num();
    for(int32 i=0;i<n;i++){
        UObject* o=g->Get(i); if(!o||!o->Class) continue;
        if(!NameIs((UObject*)o->Class,"PalCheatManager")) continue;
        if(FNameToStr(o->Name).find("Default__")!=std::string::npos) continue;
        cm=o; break;
    }
    if(!cm) return "CheatManager not found";
    uint32 f=FN_Kick->FuncFlags; FN_Kick->FuncFlags|=0x400;
    P_Kick p; new(&p.UID) FString(std::wstring(uid.begin(),uid.end()).c_str());
    PE(cm,FN_Kick,&p);
    FN_Kick->FuncFlags=f;
    return "ok";
}

// ── JSON / pipe ───────────────────────────────────────────────────────────────
static std::string jStr(const std::string& j,const char* k){
    std::string n=std::string("\"")+k+"\":\""; auto p=j.find(n);
    if(p==std::string::npos)return{};
    p+=n.size(); auto e=j.find('"',p);
    return e==std::string::npos?std::string{}:j.substr(p,e-p);
}
static int jInt(const std::string& j,const char* k){
    std::string n=std::string("\"")+k+"\":"; auto p=j.find(n);
    if(p==std::string::npos)return 1;
    p+=n.size(); while(p<j.size()&&j[p]==' ')++p;
    try{return std::stoi(j.substr(p));}catch(...){return 1;}
}
static std::string jEsc(const std::string& s){
    std::string o; for(char c:s){if(c=='"')o+="\\\"";else if(c=='\\')o+="\\\\";else o+=c;} return o;
}
static std::string Resp(const std::string& id,bool ok,const std::string& msg){
    return "{\"id\":\""+id+"\",\"ok\":"+(ok?"true":"false")+",\"msg\":\""+jEsc(msg)+"\"}\n";
}
static std::string Dispatch(const std::string& line){
    std::string id=jStr(line,"id"),cmd=jStr(line,"cmd"),result;
    try{
        if(cmd=="give_item"){
            int cnt=jInt(line,"count");
            result=DoGiveItem(jStr(line,"player_uid"),jStr(line,"item_id"),cnt>0?cnt:1);
        } else if(cmd=="kick"){
            result=DoKick(jStr(line,"user_id"));
        } else if(cmd=="ping"){
            result="pong";
        } else {
            result="unknown cmd: "+cmd;
        }
    }catch(const std::exception& e){return Resp(id,false,std::string("exception: ")+e.what());}
     catch(...){return Resp(id,false,"unknown exception");}
    return Resp(id,(result=="ok"||result=="pong"),result);
}
static void PipeServer(){
    const wchar_t* PIPE=L"\\\\.\\pipe\\pst_bridge";
    for(;;){
        HANDLE h=CreateNamedPipeW(PIPE,PIPE_ACCESS_DUPLEX,
            PIPE_TYPE_BYTE|PIPE_READMODE_BYTE|PIPE_WAIT,1,8192,8192,0,nullptr);
        if(h==INVALID_HANDLE_VALUE){Sleep(1000);continue;}
        if(!ConnectNamedPipe(h,nullptr)){CloseHandle(h);continue;}
        std::string pending; char buf[4096];
        for(;;){
            DWORD n=0;
            if(!ReadFile(h,buf,sizeof(buf)-1,&n,nullptr)||!n) break;
            buf[n]='\0'; pending+=buf;
            size_t pos;
            while((pos=pending.find('\n'))!=std::string::npos){
                std::string line=pending.substr(0,pos); pending.erase(0,pos+1);
                if(line.empty()) continue;
                std::string resp=Dispatch(line);
                DWORD w=0; WriteFile(h,resp.c_str(),(DWORD)resp.size(),&w,nullptr);
            }
        }
        DisconnectNamedPipe(h); CloseHandle(h);
    }
}

static DWORD WINAPI StartThread(LPVOID){
    g_AppendString = (AppendStringFn)(Base()+OFF_APPENDSTRING);
    g_ProcessEvent = (ProcessEventFn)(Base()+OFF_PROCESS_EVENT);
    Sleep(3000);
    PipeServer();
    return 0;
}
BOOL WINAPI DllMain(HINSTANCE hInst,DWORD reason,LPVOID){
    if(reason==DLL_PROCESS_ATTACH){
        DisableThreadLibraryCalls(hInst);
        CreateThread(nullptr,0,StartThread,nullptr,0,nullptr);
    }
    return TRUE;
}
