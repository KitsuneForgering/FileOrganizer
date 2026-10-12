import QtQuick
import QtQuick.Controls
import QtQuick.Controls.Material
import QtQuick.Layouts
import QtQuick.Dialogs

ApplicationWindow {
    id: window
    width: 1040
    height: 780
    minimumWidth: 760
    minimumHeight: 600
    visible: true
    title: "File Organizer"
    Material.theme: Material.Dark
    Material.accent: Material.Teal

    property bool previewReady: false
    property int readyCount: 0
    property string notice: "Escolha as pastas, ajuste as regras e veja a prévia."

    function dirty() {
        previewReady = false
        readyCount = 0
        moves.clear()
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

    function run(action) {
        saveProfile()
        notice = action === "preview" ? "Analisando arquivos…" : "Movendo arquivos…"
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
            } catch (e) { notice = "Configuração salva inválida; usando regras padrão." }
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
                    moves.append({ "fileName": move.name, "folderName": move.folder, "state": move.status })
                    if (move.status === "ready") readyCount++
                }
                if (command === "preview") {
                    previewReady = true
                    notice = readyCount + " arquivo(s) pronto(s); " + result.skipped + " sem regra ou ignorado(s)."
                } else {
                    previewReady = false
                    notice = result.moved + " movido(s), " + result.skipped + " ignorado(s)."
                    if (result.errors.length) notice += " Erros: " + result.errors.join("; ")
                }
            } catch (e) { notice = "Resposta inválida do organizador: " + e }
        }
        function onFailed(message) {
            notice = message || "Falha ao executar o organizador."
            previewReady = false
        }
    }

    FolderDialog {
        id: sourceDialog
        title: "Pasta de origem"
        onAccepted: { sourceField.text = decodeURIComponent(selectedFolder.toString().replace(/^file:\/\//, "")); dirty() }
    }
    FolderDialog {
        id: destinationDialog
        title: "Pasta de destino"
        onAccepted: { destinationField.text = decodeURIComponent(selectedFolder.toString().replace(/^file:\/\//, "")); dirty() }
    }

    ListModel {
        id: ruleModel
        ListElement { extensions: "pdf, doc, docx, odt, txt"; folder: "Documentos"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "jpg, jpeg, png, webp, gif, svg"; folder: "Imagens"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "mp4, mkv, mov, webm"; folder: "Vídeos"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "mp3, flac, ogg, wav"; folder: "Músicas"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
        ListElement { extensions: "zip, tar, gz, 7z, rar"; folder: "Arquivos"; exactName: ""; nameContains: ""; nameGlob: ""; nameRegex: ""; nameFuzzy: ""; maxEdits: ""; mimeTypes: ""; minBytes: ""; maxBytes: "" }
    }
    ListModel { id: moves }

    Dialog {
        id: confirmDialog
        title: "Mover arquivos?"
        modal: true
        anchors.centerIn: parent
        standardButtons: Dialog.Yes | Dialog.Cancel
        onAccepted: run("apply")
        contentItem: Label {
            text: "Mover " + readyCount + " arquivo(s) de acordo com as regras atuais?\nA pasta será verificada novamente antes da operação."
            wrapMode: Text.WordWrap
            padding: 16
        }
    }

    ScrollView {
        anchors.fill: parent
        anchors.margins: 22
        contentWidth: availableWidth
        ColumnLayout {
            width: parent.width
            spacing: 16

            ColumnLayout {
                Layout.fillWidth: true
                spacing: 4
                Label { text: "File Organizer"; font.pixelSize: 30; font.bold: true }
                Label { text: "Organização por regras com prévia antes de mover"; opacity: 0.7 }
            }

            Frame {
                Layout.fillWidth: true
                ColumnLayout {
                    anchors.fill: parent
                    spacing: 10
                    Label { text: "PASTAS"; font.bold: true; opacity: 0.8 }
                    RowLayout {
                        Layout.fillWidth: true
                        Label { text: "Origem"; Layout.preferredWidth: 80 }
                        TextField {
                            id: sourceField
                            Layout.fillWidth: true
                            placeholderText: "Ex.: /home/usuário/Downloads"
                            onTextEdited: dirty()
                        }
                        Button { text: "Escolher"; onClicked: sourceDialog.open() }
                    }
                    RowLayout {
                        Layout.fillWidth: true
                        Label { text: "Destino"; Layout.preferredWidth: 80 }
                        TextField {
                            id: destinationField
                            Layout.fillWidth: true
                            placeholderText: "Ex.: /home/usuário/Organizados"
                            onTextEdited: dirty()
                        }
                        Button { text: "Escolher"; onClicked: destinationDialog.open() }
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
                        Label { text: "REGRAS"; font.bold: true; opacity: 0.8; Layout.fillWidth: true }
                        Button {
                            text: "+ Adicionar regra"
                            onClicked: {
                                ruleModel.append({ "extensions": "", "folder": "", "exactName": "", "nameContains": "",
                                                   "nameGlob": "", "nameRegex": "", "nameFuzzy": "", "maxEdits": "",
                                                   "mimeTypes": "", "minBytes": "", "maxBytes": "" })
                                dirty()
                            }
                        }
                    }
                    Label {
                        text: "A primeira regra correspondente vence. Campos preenchidos são combinados (E); valores separados por vírgula usam OU."
                        opacity: 0.65
                        wrapMode: Text.WordWrap
                        Layout.fillWidth: true
                    }
                    Repeater {
                        model: ruleModel
                        delegate: Frame {
                            required property int index
                            required property string extensions
                            required property string folder
                            required property string exactName
                            required property string nameContains
                            required property string nameGlob
                            required property string nameRegex
                            required property string nameFuzzy
                            required property string maxEdits
                            required property string mimeTypes
                            required property string minBytes
                            required property string maxBytes
                            property bool expanded: false
                            Layout.fillWidth: true
                            ColumnLayout {
                                anchors.fill: parent
                                spacing: 6
                                RowLayout {
                                    Layout.fillWidth: true
                                    Label { text: (index + 1) + "."; Layout.preferredWidth: 28 }
                                    TextField {
                                        Layout.fillWidth: true
                                        text: extensions
                                        placeholderText: "Extensões: pdf, docx (opcional)"
                                        onTextEdited: { ruleModel.setProperty(index, "extensions", text); dirty() }
                                    }
                                    Label { text: "→" }
                                    TextField {
                                        Layout.preferredWidth: 210
                                        text: folder
                                        placeholderText: "Pasta de destino"
                                        onTextEdited: { ruleModel.setProperty(index, "folder", text); dirty() }
                                    }
                                    ToolButton {
                                        text: "×"
                                        ToolTip.visible: hovered
                                        ToolTip.text: "Remover regra"
                                        onClicked: { ruleModel.remove(index); dirty() }
                                    }
                                }
                                Button {
                                    text: expanded ? "Ocultar filtros" : "Nome, aproximação, MIME e tamanho"
                                    flat: true
                                    onClicked: expanded = !expanded
                                }
                                GridLayout {
                                    visible: expanded
                                    columns: 2
                                    columnSpacing: 10
                                    rowSpacing: 4
                                    Layout.fillWidth: true
                                    Label { text: "Nome exato" }
                                    TextField {
                                        Layout.fillWidth: true
                                        text: exactName
                                        placeholderText: "Relatório.pdf"
                                        onTextEdited: { ruleModel.setProperty(index, "exactName", text); dirty() }
                                    }
                                    Label { text: "Padrão glob" }
                                    TextField {
                                        Layout.fillWidth: true
                                        text: nameGlob
                                        placeholderText: "fatura-*.pdf"
                                        onTextEdited: { ruleModel.setProperty(index, "nameGlob", text); dirty() }
                                    }
                                    Label { text: "Nome contém" }
                                    TextField {
                                        Layout.fillWidth: true
                                        text: nameContains
                                        placeholderText: "fatura (ignora maiúsculas)"
                                        onTextEdited: { ruleModel.setProperty(index, "nameContains", text); dirty() }
                                    }
                                    Label { text: "Regex do nome" }
                                    TextField {
                                        Layout.fillWidth: true
                                        text: nameRegex
                                        placeholderText: "^IMG_[0-9]+\\.jpg$"
                                        onTextEdited: { ruleModel.setProperty(index, "nameRegex", text); dirty() }
                                    }
                                    Label { text: "Nome aproximado" }
                                    RowLayout {
                                        Layout.fillWidth: true
                                        TextField {
                                            Layout.fillWidth: true
                                            text: nameFuzzy
                                            placeholderText: "relatorio.pdf"
                                            onTextEdited: {
                                                ruleModel.setProperty(index, "nameFuzzy", text)
                                                if (text.length === 0) ruleModel.setProperty(index, "maxEdits", "")
                                                dirty()
                                            }
                                        }
                                        Label { text: "Diferenças até" }
                                        ComboBox {
                                            model: ["1", "2", "3"]
                                            enabled: nameFuzzy.length > 0
                                            currentIndex: maxEdits === "" ? 0 : Number(maxEdits) - 1
                                            onActivated: { ruleModel.setProperty(index, "maxEdits", currentText); dirty() }
                                        }
                                    }
                                    Label { text: "Tipos MIME" }
                                    TextField {
                                        Layout.fillWidth: true
                                        text: mimeTypes
                                        placeholderText: "image/*, application/pdf"
                                        onTextEdited: { ruleModel.setProperty(index, "mimeTypes", text); dirty() }
                                    }
                                    Label { text: "Tamanho (bytes)" }
                                    RowLayout {
                                        Layout.fillWidth: true
                                        TextField {
                                            Layout.fillWidth: true
                                            text: minBytes
                                            placeholderText: "Mínimo"
                                            validator: RegularExpressionValidator { regularExpression: /[0-9]*/ }
                                            onTextEdited: { ruleModel.setProperty(index, "minBytes", text); dirty() }
                                        }
                                        Label { text: "até" }
                                        TextField {
                                            Layout.fillWidth: true
                                            text: maxBytes
                                            placeholderText: "Máximo"
                                            validator: RegularExpressionValidator { regularExpression: /[0-9]*/ }
                                            onTextEdited: { ruleModel.setProperty(index, "maxBytes", text); dirty() }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }

            RowLayout {
                Layout.fillWidth: true
                Button { text: "Ver prévia"; enabled: !organizer.busy; onClicked: run("preview") }
                Button {
                    text: "Mover arquivos"
                    highlighted: true
                    enabled: previewReady && readyCount > 0 && !organizer.busy
                    onClicked: confirmDialog.open()
                }
                BusyIndicator { running: organizer.busy; visible: running; Layout.preferredWidth: 36; Layout.preferredHeight: 36 }
                Item { Layout.fillWidth: true }
                Label { text: readyCount + " pronto(s)"; opacity: 0.7 }
            }

            Frame {
                Layout.fillWidth: true
                Layout.preferredHeight: 250
                ColumnLayout {
                    anchors.fill: parent
                    Label { text: "PRÉVIA / RESULTADO"; font.bold: true; opacity: 0.8 }
                    Label { text: notice; Layout.fillWidth: true; wrapMode: Text.WordWrap }
                    ListView {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        clip: true
                        model: moves
                        delegate: RowLayout {
                            required property string fileName
                            required property string folderName
                            required property string state
                            width: ListView.view.width
                            height: 34
                            Label { text: fileName; Layout.fillWidth: true; elide: Text.ElideRight }
                            Label { text: "→ " + folderName; Layout.preferredWidth: 230; elide: Text.ElideRight; opacity: 0.8 }
                            Label {
                                text: state === "ready" ? "Pronto" : state === "moved" ? "Movido" : state === "conflict" ? "Já existe" : "Erro"
                                color: state === "conflict" || state === "error" ? "#ffb86c" : "#71d9b0"
                                Layout.preferredWidth: 85
                            }
                        }
                    }
                }
            }
            Label {
                text: "Somente arquivos regulares da pasta de origem. Subpastas e arquivos ocultos são ignorados; conflitos nunca sobrescrevem."
                opacity: 0.55
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
        }
    }
}
