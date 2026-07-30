#pragma once
#ifdef __cplusplus
extern "C" {
#endif
int seh_call_appendstring(void* fn, const void* name, void* fstr);
int seh_call_processevent(void* fn, void* obj, void* func, void* parms);
#ifdef __cplusplus
}
#endif
