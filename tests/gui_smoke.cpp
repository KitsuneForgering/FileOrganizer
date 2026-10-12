#include "OrganizerBridge.h"

#include <QFile>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QSignalSpy>
#include <QTemporaryDir>
#include <QTest>

class GuiSmokeTest : public QObject {
    Q_OBJECT
private slots:
    void loadsWindow() {
        OrganizerBridge bridge;
        QQmlApplicationEngine engine;
        engine.rootContext()->setContextProperty("organizer", &bridge);
        engine.load(QUrl::fromLocalFile(QStringLiteral(FILEORGANIZER_QML_PATH)));
        QCOMPARE(engine.rootObjects().size(), 1);
        QVERIFY(engine.rootObjects().first()->property("visible").toBool());
        QVariant payload;
        QVERIFY(QMetaObject::invokeMethod(engine.rootObjects().first(), "request", Q_RETURN_ARG(QVariant, payload)));
        const auto profile = QJsonDocument::fromJson(payload.toString().toUtf8()).object();
        const auto rules = profile.value("rules").toArray();
        QVERIFY(!rules.isEmpty());
        QVERIFY(rules.first().toObject().contains("name_regex"));
        QVERIFY(rules.first().toObject().contains("name_contains"));
        QVERIFY(rules.first().toObject().contains("name_fuzzy"));
        QVERIFY(rules.first().toObject().contains("mime_types"));
    }

    void bridgeRunsCore() {
        QTemporaryDir temp;
        QVERIFY(temp.isValid());
        QDir root(temp.path());
        QVERIFY(root.mkdir("source"));
        QFile file(root.filePath("source/note.txt"));
        QVERIFY(file.open(QIODevice::WriteOnly));
        QCOMPARE(file.write("hello"), 5);
        file.close();

        const QJsonObject request{
            {"source", root.filePath("source")},
            {"destination", root.filePath("dest")},
            {"rules", QJsonArray{QJsonObject{{"extensions", QJsonArray{"txt"}}, {"folder", "Notes"}}}}
        };
        OrganizerBridge bridge;
        QSignalSpy completed(&bridge, &OrganizerBridge::completed);
        QSignalSpy failed(&bridge, &OrganizerBridge::failed);
        bridge.run("preview", QString::fromUtf8(QJsonDocument(request).toJson(QJsonDocument::Compact)));
        QVERIFY2(completed.wait(5000), "Go preview did not complete");
        QCOMPARE(failed.size(), 0);
        const auto result = QJsonDocument::fromJson(completed.first().at(1).toString().toUtf8()).object();
        const auto moves = result.value("moves").toArray();
        QCOMPARE(moves.size(), 1);
        QCOMPARE(moves.first().toObject().value("status").toString(), QStringLiteral("ready"));
        QVERIFY(file.exists());

        completed.clear();
        bridge.run("apply", QString::fromUtf8(QJsonDocument(request).toJson(QJsonDocument::Compact)));
        QVERIFY2(completed.wait(5000), "Go apply did not complete");
        QCOMPARE(failed.size(), 0);
        const auto applied = QJsonDocument::fromJson(completed.first().at(1).toString().toUtf8()).object();
        QCOMPARE(applied.value("moved").toInt(), 1);
        QVERIFY(!file.exists());
        QVERIFY(QFile::exists(root.filePath("dest/Notes/note.txt")));
    }
};

QTEST_MAIN(GuiSmokeTest)
#include "gui_smoke.moc"
