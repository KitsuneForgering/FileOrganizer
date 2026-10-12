#pragma once

#include <QObject>
#include <QProcess>
#include <QSettings>

class OrganizerBridge : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
public:
    explicit OrganizerBridge(QObject *parent = nullptr);
    bool busy() const { return process.state() != QProcess::NotRunning; }
    Q_INVOKABLE void run(const QString &command, const QString &requestJson);
    Q_INVOKABLE QString loadSettings() const;
    Q_INVOKABLE void saveSettings(const QString &json);
signals:
    void busyChanged();
    void completed(const QString &command, const QString &resultJson);
    void failed(const QString &message);
private:
    QProcess process;
    QSettings settings;
    QString command;
};
