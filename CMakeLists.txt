cmake_minimum_required(VERSION 3.21)
project(FileOrganizer LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY "${CMAKE_SOURCE_DIR}/build/bin")
find_package(Qt6 6.5 REQUIRED COMPONENTS Quick QuickControls2 Test)
qt_standard_project_setup()
qt_policy(SET QTP0001 NEW)
qt_policy(SET QTP0004 NEW)

qt_add_executable(fileorganizer gui/main.cpp gui/OrganizerBridge.cpp)
set_source_files_properties(gui/Main.qml PROPERTIES QT_RESOURCE_ALIAS Main.qml)
qt_add_qml_module(fileorganizer
    URI FileOrganizer
    VERSION 1.0
    QML_FILES gui/Main.qml
)
target_link_libraries(fileorganizer PRIVATE Qt6::Quick Qt6::QuickControls2)

enable_testing()
qt_add_executable(fileorganizer-gui-test tests/gui_smoke.cpp gui/OrganizerBridge.cpp)
target_include_directories(fileorganizer-gui-test PRIVATE gui)
target_compile_definitions(fileorganizer-gui-test PRIVATE FILEORGANIZER_QML_PATH="${CMAKE_SOURCE_DIR}/gui/Main.qml")
target_link_libraries(fileorganizer-gui-test PRIVATE Qt6::Quick Qt6::QuickControls2 Qt6::Test)
add_test(NAME gui-smoke COMMAND fileorganizer-gui-test)
set_tests_properties(gui-smoke PROPERTIES ENVIRONMENT "QT_QPA_PLATFORM=offscreen;QT_QPA_PLATFORMTHEME=;QT_QUICK_BACKEND=software;GTK_MODULES=;XDG_CONFIG_HOME=${CMAKE_BINARY_DIR}/test-config")
