cmake_minimum_required(VERSION 3.20)
project(arbeiter_auth_sdk VERSION 0.1.0 LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(CURL REQUIRED)
find_package(nlohmann_json 3.11 REQUIRED)

add_library(arbeiter_auth src/client.cpp src/c_api.cpp)
target_include_directories(arbeiter_auth PUBLIC ${CMAKE_CURRENT_SOURCE_DIR}/include)
target_link_libraries(arbeiter_auth PUBLIC CURL::libcurl nlohmann_json::nlohmann_json)

include(CTest)
if(BUILD_TESTING)
  add_executable(arbeiter_auth_sdk_test ../test/cpp/client_test.cpp ../test/cpp/c_api_smoke.c)
  set_target_properties(arbeiter_auth_sdk_test PROPERTIES LINKER_LANGUAGE CXX)
  target_link_libraries(arbeiter_auth_sdk_test PRIVATE arbeiter_auth)
  add_test(NAME arbeiter_auth_sdk_test COMMAND arbeiter_auth_sdk_test)
endif()
