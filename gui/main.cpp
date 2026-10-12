#include "OrganizerBridge.h"

#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>

int main(int argc, char *argv[]) {
    QGuiApplication app(argc, argv);
    OrganizerBridge bridge;
    QQmlApplicationEngine engine;
    engine.rootContext()->setContextProperty("organizer", &bridge);
    engine.loadFromModule("FileOrganizer", "Main");
    if (engine.rootObjects().isEmpty()) return 1;
    return app.exec();
}
