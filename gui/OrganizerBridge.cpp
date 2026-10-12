#include "OrganizerBridge.h"

#include <QCoreApplication>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QSaveFile>
#include <QStandardPaths>

namespace {
QString profilePath() {
    return QDir(QStandardPaths::writableLocation(QStandardPaths::ConfigLocation))
        .filePath("fileorganizer/profile.json");
}
}

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
            emit failed("Could not start fileorganizer-core. Run make build.");
        }
    });
}

void OrganizerBridge::run(const QString &action, const QString &requestJson) {
    if (busy()) return;
    if (action != "preview" && action != "apply") { emit failed("Invalid action."); return; }
    command = action;
    const QString backend = QDir(QCoreApplication::applicationDirPath()).filePath("fileorganizer-core");
    if (!QFileInfo::exists(backend)) { emit failed("fileorganizer-core was not found. Run make build."); return; }
    process.start(backend, {action});
    if (!process.waitForStarted(3000)) return;
    process.write(requestJson.toUtf8());
    process.closeWriteChannel();
}

QString OrganizerBridge::loadSettings() const {
    QFile profile(profilePath());
    if (profile.exists()) {
        if (!profile.open(QIODevice::ReadOnly)) return {};
        return QString::fromUtf8(profile.readAll());
    }
    return settings.value("profile/json").toString();
}

void OrganizerBridge::saveSettings(const QString &json) {
    const QString path = profilePath();
    if (!QDir().mkpath(QFileInfo(path).absolutePath())) {
        emit failed("Could not create the configuration directory.");
        return;
    }
    QSaveFile profile(path);
    const QByteArray data = json.toUtf8();
    if (!profile.open(QIODevice::WriteOnly) || profile.write(data) != data.size() || !profile.commit()) {
        emit failed("Could not save the rules to " + path);
    }
}
