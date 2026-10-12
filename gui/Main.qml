import QtQuick
import QtQuick.Controls
import QtQuick.Controls.Material
import QtQuick.Layouts
import QtQuick.Dialogs

ApplicationWindow {
    id: window
    width: 760
    height: 680
    minimumWidth: 580
    minimumHeight: 520
    visible: true
    title: "File Organizer"
    Material.theme: Material.Dark
    Material.accent: Material.Teal

    property bool previewReady: false
    property int readyCount: 0
    property string notice: "Choose folders, review the rules, then preview the moves."

    function dirty() {
        previewReady = false
        readyCount = 0
        moves.clear()
        organizer.saveSettings(request())
    }

    function request() {
        var rules = []
        for (var i = 0; i < ruleModel.count; ++i) {
            var rule = ruleModel.get(i)
            var extensions = rule.extensions.split(",").map(function(s) { return s.trim() }).filter(function(s) { return s.length > 0 })
            var mimeTypes = rule.mimeTypes.split(",").map(function(s) { return s.trim() }).filter(function(s) { return s.length > 0 })
            rules.push({ "extensions": extensions,
                         "name": rule.exactName,
                         "name_contains": rule.nameContains,
                         "name_glob": rule.nameGlob,
                         "name_regex": rule.nameRegex,
                         "name_fuzzy": rule.nameFuzzy,
                         "max_edits": rule.maxEdits === "" ? null : Number(rule.maxEdits),
                         "mime_types": mimeTypes,
                         "min_bytes": rule.minBytes === "" ? null : Number(rule.minBytes),
                         "max_bytes": rule.maxBytes === "" ? null : Number(rule.maxBytes),
                         "folder": rule.folder.trim() })
        }
        return JSON.stringify({ "source": sourceField.text.trim(),
                                "destination": destinationField.text.trim(), "rules": rules })
    }

    function saveProfile() {
        organizer.saveSettings(request())
    }

    function ruleSummary(rule) {
        var parts = []
        if (rule.extensions) parts.push(rule.extensions)
        if (rule.exactName) parts.push("name = " + rule.exactName)
        if (rule.nameContains) parts.push("contains " + rule.nameContains)
        if (rule.nameGlob) parts.push("glob " + rule.nameGlob)
        if (rule.nameRegex) parts.push("regex " + rule.nameRegex)
        if (rule.nameFuzzy) parts.push("similar to " + rule.nameFuzzy)
        if (rule.mimeTypes) parts.push("MIME " + rule.mimeTypes)
        if (rule.minBytes || rule.maxBytes) parts.push("size")
        return (parts.length ? parts.join(" + ") : "Set a condition") + "  →  " + (rule.folder || "Choose a folder")
    }

    function run(action) {
        saveProfile()
        notice = action === "preview" ? "Scanning files…" : "Moving files…"
        organizer.run(action, request())
    }

    Component.onCompleted: {
        var saved = organizer.loadSettings()
        if (saved.length > 0) {
            try {
                var profile = JSON.parse(saved)
                sourceField.text = profile.source || ""
                destinationField.text = profile.destination || ""
                if (profile.rules && profile.rules.length) {
                    ruleModel.clear()
                    for (var i = 0; i < profile.rules.length; ++i) {
                        var rule = profile.rules[i]
                        ruleModel.append({ "extensions": (rule.extensions || []).join(", "),
                                           "exactName": rule.name || "",
                                           "nameContains": rule.name_contains || "",
                                           "nameGlob": rule.name_glob || "",
                                           "nameRegex": rule.name_regex || "",
                                           "nameFuzzy": rule.name_fuzzy || "",
                                           "maxEdits": rule.max_edits === undefined || rule.max_edits === null ? "" : String(rule.max_edits),
                                           "mimeTypes": (rule.mime_types || []).join(", "),
                                           "minBytes": rule.min_bytes === undefined || rule.min_bytes === null ? "" : String(rule.min_bytes),
                                           "maxBytes": rule.max_bytes === undefined || rule.max_bytes === null ? "" : String(rule.max_bytes),
                                           "folder": rule.folder || "" })
                    }
                }
            } catch (e) { notice = "Saved profile is invalid; using default rules." }
        }
    }

    Connections {
        target: organizer
        function onCompleted(command, resultJson) {
            try {
                var result = JSON.parse(resultJson)
                moves.clear()
                readyCount = 0
                for (var i = 0; i < result.moves.length; ++i) {
                    var move = result.moves[i]
                    moves.append({ "fileName": move.name, "folderName": move.folder, "moveStatus": move.status })
                    if (move.status === "ready") readyCount++
                }
                if (command === "preview") {
                    previewReady = true
                    notice = readyCount + " file(s) ready; " + result.skipped + " unmatched or skipped."
                } else {
                    previewReady = false
                    notice = result.moved + " moved, " + result.skipped + " skipped."
                    if (result.errors.length) notice += " Errors: " + result.errors.join("; ")
                }
            } catch (e) { notice = "Invalid organizer response: " + e }
        }
        function onFailed(message) {
            notice = message || "Could not run the organizer."
            previewReady = false
        }
    }

    FolderDialog {
        id: sourceDialog
        title: "Source folder"
        onAccepted: { sourceField.text = decodeURIComponent(selectedFolder.toString().replace(/^file:\/\//, "")); dirty() }
    }
    FolderDialog {
        id: destinationDialog
        title: "Destination folder"
        onAccepted: { destinationField.text = decodeURIComponent(selectedFolder.toString().replace(/^file:\/\//, "")); dirty() }
    }

    ListModel {
        id: ruleModel
        ListElement { extensions: "pdf, doc, docx, odt, txt"; folder: "Documents"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "jpg, jpeg, png, webp, gif, svg"; folder: "Images"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "mp4, mkv, mov, webm"; folder: "Videos"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "mp3, flac, ogg, wav"; folder: "Music"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "zip, tar, gz, 7z, rar"; folder: "Archives"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
    }
    ListModel { id: moves }

    Dialog {
        id: confirmDialog
        title: "Move files?"
        width: Math.min(window.width - 40, 440)
        modal: true
        anchors.centerIn: parent
        standardButtons: Dialog.Yes | Dialog.Cancel
        onAccepted: run("apply")
        contentItem: Label {
            text: "Move " + readyCount + " file(s) using the current rules?\nThe source folder will be checked again before moving."
            wrapMode: Text.WordWrap
            padding: 16
        }
    }

    Dialog {
        id: ruleDialog
        property int ruleIndex: -1
        property bool showAdvanced: false
        title: ruleIndex < 0 ? "New rule" : "Edit rule"
        width: Math.min(window.width - 40, 620)
        height: Math.min(window.height - 40, 600)
        anchors.centerIn: parent
        modal: true
        standardButtons: Dialog.Ok | Dialog.Cancel

        function openFor(index) {
            ruleIndex = index
            var rule = index < 0 ? null : ruleModel.get(index)
            showAdvanced = !!(rule && (rule.exactName || rule.nameContains || rule.nameGlob ||
                                       rule.nameRegex || rule.nameFuzzy || rule.mimeTypes ||
                                       rule.minBytes || rule.maxBytes))
            editExtensions.text = rule ? rule.extensions : ""
            editFolder.text = rule ? rule.folder : ""
            editExactName.text = rule ? rule.exactName : ""
            editNameContains.text = rule ? rule.nameContains : ""
            editNameGlob.text = rule ? rule.nameGlob : ""
            editNameRegex.text = rule ? rule.nameRegex : ""
            editNameFuzzy.text = rule ? rule.nameFuzzy : ""
            editMaxEdits.currentIndex = rule && rule.maxEdits !== "" ? Number(rule.maxEdits) - 1 : 0
            editMimeTypes.text = rule ? rule.mimeTypes : ""
            editMinBytes.text = rule ? rule.minBytes : ""
            editMaxBytes.text = rule ? rule.maxBytes : ""
            open()
        }

        onAccepted: {
            var entry = { "extensions": editExtensions.text.trim(), "folder": editFolder.text.trim(),
                          "exactName": editExactName.text.trim(), "nameContains": editNameContains.text.trim(),
                          "nameGlob": editNameGlob.text.trim(), "nameRegex": editNameRegex.text.trim(),
                          "nameFuzzy": editNameFuzzy.text.trim(),
                          "maxEdits": editNameFuzzy.text.trim() ? editMaxEdits.currentText : "",
                          "mimeTypes": editMimeTypes.text.trim(), "minBytes": editMinBytes.text.trim(),
                          "maxBytes": editMaxBytes.text.trim() }
            if (ruleIndex < 0) ruleModel.append(entry)
            else ruleModel.set(ruleIndex, entry)
            dirty()
        }

        contentItem: ScrollView {
            clip: true
            contentWidth: availableWidth
            ColumnLayout {
                width: parent.width
                spacing: 10
                Label { text: "When a file matches, move it to:"; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                TextField { id: editFolder; Layout.fillWidth: true; placeholderText: "Folder, for example: Documents" }
                Label { text: "Extensions" }
                TextField { id: editExtensions; Layout.fillWidth: true; placeholderText: "pdf, docx" }
                Label { text: "Separate extensions with commas. The first matching rule wins."; opacity: 0.7; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                Button {
                    text: ruleDialog.showAdvanced ? "Hide other conditions" : "Other conditions: name, MIME, size…"
                    flat: true
                    onClicked: ruleDialog.showAdvanced = !ruleDialog.showAdvanced
                }
                ColumnLayout {
                    visible: ruleDialog.showAdvanced
                    Layout.fillWidth: true
                    spacing: 8
                    Label { text: "Filled conditions are combined. Leave the others empty."; opacity: 0.7; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                    Label { text: "Name contains" }
                    TextField { id: editNameContains; Layout.fillWidth: true; placeholderText: "invoice" }
                    Label { text: "Exact name" }
                    TextField { id: editExactName; Layout.fillWidth: true; placeholderText: "Report.pdf" }
                    Label { text: "Glob pattern" }
                    TextField { id: editNameGlob; Layout.fillWidth: true; placeholderText: "invoice-*.pdf" }
                    Label { text: "Name regex" }
                    TextField { id: editNameRegex; Layout.fillWidth: true; placeholderText: "^IMG_[0-9]+\\.jpg$" }
                    Label { text: "Similar name" }
                    RowLayout {
                        Layout.fillWidth: true
                        TextField { id: editNameFuzzy; Layout.fillWidth: true; placeholderText: "report.pdf" }
                        Label { text: "Differences" }
                        ComboBox { id: editMaxEdits; model: ["1", "2", "3"]; enabled: editNameFuzzy.text.length > 0 }
                    }
                    Label { text: "MIME types" }
                    TextField { id: editMimeTypes; Layout.fillWidth: true; placeholderText: "image/*, application/pdf" }
                    Label { text: "Size in bytes" }
                    RowLayout {
                        Layout.fillWidth: true
                        TextField { id: editMinBytes; Layout.fillWidth: true; placeholderText: "Minimum"; validator: RegularExpressionValidator { regularExpression: /[0-9]*/ } }
                        Label { text: "to" }
                        TextField { id: editMaxBytes; Layout.fillWidth: true; placeholderText: "Maximum"; validator: RegularExpressionValidator { regularExpression: /[0-9]*/ } }
                    }
                }
            }
        }
    }

    ScrollView {
        anchors.fill: parent
        anchors.margins: 24
        contentWidth: availableWidth
        ColumnLayout {
            width: parent.width
            spacing: 18

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4
                Label { text: "Organize files"; font.pixelSize: 28; font.bold: true }
                Label { text: "Choose folders and check the result before moving."; opacity: 0.7; wrapMode: Text.WordWrap; Layout.fillWidth: true }
            }

            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    spacing: 10
                    Label { text: "1  Folders"; font.pixelSize: 18; font.bold: true }
                    Label { text: "Files in"; opacity: 0.8 }
                    RowLayout {
                        Layout.fillWidth: true
                        TextField {
                            id: sourceField
                            Layout.fillWidth: true
                            placeholderText: "Choose the folder to organize"
                            onTextEdited: dirty()
                        }
                        Button { text: "Browse…"; onClicked: sourceDialog.open() }
                    }
                    Label { text: "Move to"; opacity: 0.8 }
                    RowLayout {
                        Layout.fillWidth: true
                        TextField {
                            id: destinationField
                            Layout.fillWidth: true
                            placeholderText: "Choose where files should go"
                            onTextEdited: dirty()
                        }
                        Button { text: "Browse…"; onClicked: destinationDialog.open() }
                    }
                }
            }

            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    spacing: 8
                    RowLayout {
                        Layout.fillWidth: true
                        Label { text: "2  Rules"; font.pixelSize: 18; font.bold: true; Layout.fillWidth: true }
                        Button { text: "Add"; onClicked: ruleDialog.openFor(-1) }
                    }
                    Repeater {
                        model: ruleModel
                        delegate: RowLayout {
                            required property int index
                            required property string extensions
                            required property string folder
                            required property string exactName
                            required property string nameContains
                            required property string nameGlob
                            required property string nameRegex
                            required property string nameFuzzy
                            required property string mimeTypes
                            required property string minBytes
                            required property string maxBytes
                            Layout.fillWidth: true
                            spacing: 8
                            Label {
                                HoverHandler { id: ruleHover }
                                text: (index + 1) + ".  " + ruleSummary({ "extensions": extensions, "folder": folder,
                                    "exactName": exactName, "nameContains": nameContains, "nameGlob": nameGlob,
                                    "nameRegex": nameRegex, "nameFuzzy": nameFuzzy, "mimeTypes": mimeTypes,
                                    "minBytes": minBytes, "maxBytes": maxBytes })
                                Layout.fillWidth: true
                                elide: Text.ElideRight
                                ToolTip.visible: ruleHover.hovered
                                ToolTip.text: text
                            }
                            Button { text: "Edit"; flat: true; onClicked: ruleDialog.openFor(index) }
                            ToolButton {
                                text: "×"
                                ToolTip.visible: hovered
                                ToolTip.text: "Remove rule"
                                onClicked: { ruleModel.remove(index); dirty() }
                            }
                        }
                    }
                    Label {
                        text: "The first matching rule chooses the destination folder."
                        opacity: 0.65
                        wrapMode: Text.WordWrap
                        Layout.fillWidth: true
                    }
                }
            }

            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    spacing: 10
                    Label { text: "3  Preview and move"; font.pixelSize: 18; font.bold: true }
                    RowLayout {
                        Layout.fillWidth: true
                        Button {
                            text: "Preview"
                            enabled: !organizer.busy
                            onClicked: run("preview")
                        }
                        Button {
                            text: "Move " + readyCount + " file(s)"
                            highlighted: true
                            visible: previewReady && readyCount > 0
                            enabled: !organizer.busy
                            onClicked: confirmDialog.open()
                        }
                        BusyIndicator { running: organizer.busy; visible: running; Layout.preferredWidth: 32; Layout.preferredHeight: 32 }
                    }
                    Label { text: notice; wrapMode: Text.WordWrap; Layout.fillWidth: true }
                    ListView {
                        visible: moves.count > 0
                        Layout.fillWidth: true
                        Layout.preferredHeight: Math.min(moves.count * 38, 220)
                        clip: true
                        model: moves
                        delegate: RowLayout {
                            required property string fileName
                            required property string folderName
                            required property string moveStatus
                            width: ListView.view.width
                            height: 36
                            Label { text: fileName; Layout.fillWidth: true; elide: Text.ElideRight }
                            Label { text: "→ " + folderName; Layout.preferredWidth: 160; elide: Text.ElideRight; opacity: 0.8 }
                            Label {
                                text: moveStatus === "ready" ? "Ready" : moveStatus === "moved" ? "Moved" : moveStatus === "conflict" ? "Exists" : "Error"
                                color: moveStatus === "conflict" || moveStatus === "error" ? "#ffb86c" : "#71d9b0"
                                Layout.preferredWidth: 75
                            }
                        }
                    }
                }
            }
        }
    }
}
