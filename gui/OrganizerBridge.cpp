#include "OrganizerBridge.h"

#include <QCoreApplication>
#include <QDir>
#include <QFileInfo>

OrganizerBridge::OrganizerBridge(QObject *parent)
    : QObject(parent), settings(QSettings::IniFormat, QSettings::UserScope,
                                "FileOrganizer", "FileOrganizer") {
    connect(&process, &QProcess::started, this, &OrganizerBridge::busyChanged);
    connect(&process, &QProcess::finished, this, [this](int exitCode) {
        emit busyChanged();
        if (exitCode == 0) {
            emit completed(command, QString::fromUtf8(process.readAllStandardOutput()));
        } else {
            emit failed(QString::fromUtf8(process.readAllStandardError()).trimmed());
        }
    });
    connect(&process, &QProcess::errorOccurred, this, [this](QProcess::ProcessError error) {
        if (error == QProcess::FailedToStart) {
            emit busyChanged();
            emit failed("Não foi possível iniciar fileorganizer-core. Execute make build.");
        }
    });
}

void OrganizerBridge::run(const QString &action, const QString &requestJson) {
    if (busy()) return;
    if (action != "preview" && action != "apply") { emit failed("Ação inválida."); return; }
    command = action;
    const QString backend = QDir(QCoreApplication::applicationDirPath()).filePath("fileorganizer-core");
    if (!QFileInfo::exists(backend)) { emit failed("fileorganizer-core não encontrado. Execute make build."); return; }
    process.start(backend, {action});
    if (!process.waitForStarted(3000)) return;
    process.write(requestJson.toUtf8());
    process.closeWriteChannel();
}

QString OrganizerBridge::loadSettings() const {
    return settings.value("profile/json").toString();
}

void OrganizerBridge::saveSettings(const QString &json) {
    settings.setValue("profile/json", json);
}
