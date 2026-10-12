import QtQuick
import qs.Ui

BarWidget {
  id: root
  moduleName: "kitsuneforgering.fileorganizer"
  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    text: "Files"
    horizontalMargin: 8
    onPressed: function(mouseButton) {
      if (mouseButton === Qt.LeftButton && root.bar)
        root.bar.run("fileorganizer")
    }
  }
}
