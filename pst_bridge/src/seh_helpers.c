/* seh_helpers.c - plain C SEH wrappers, no C++ destructors */
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include "seh_helpers.h"

int seh_call_appendstring(void* fn, const void* name, void* fstr) {
    __try {
        ((void(*)(const void*, void*))fn)(name, fstr);
        return 1;
    } __except(EXCEPTION_EXECUTE_HANDLER) { return 0; }
}

int seh_call_processevent(void* fn, void* obj, void* func, void* parms) {
    __try {
        ((void(*)(void*, void*, void*))fn)(obj, func, parms);
        return 1;
    } __except(EXCEPTION_EXECUTE_HANDLER) { return 0; }
}
