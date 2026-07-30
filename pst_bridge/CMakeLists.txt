cmake_minimum_required(VERSION 3.20)
project(pst_bridge CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Release build strips debug info and optimises
if(NOT CMAKE_BUILD_TYPE)
  set(CMAKE_BUILD_TYPE Release)
endif()

add_library(pst_bridge SHARED src/dllmain.cpp)

target_compile_options(pst_bridge PRIVATE
  $<$<CXX_COMPILER_ID:MSVC>:/O2 /GS- /EHsc>
)

target_link_libraries(pst_bridge PRIVATE
  kernel32
)

set_target_properties(pst_bridge PROPERTIES
  OUTPUT_NAME "pst_bridge"
  PREFIX ""
)
