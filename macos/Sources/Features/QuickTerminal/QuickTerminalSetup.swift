import AppKit
import ApplicationServices
import SwiftUI

/// A small floating guide for setting up the Quick Terminal's global shortcut.
final class QuickTerminalSetupWindowController: NSWindowController, NSWindowDelegate {
    private weak var appDelegate: AppDelegate?
    private var refreshTimer: Timer?

    init(appDelegate: AppDelegate) {
        self.appDelegate = appDelegate

        let panel = QuickTerminalSetupPanel(
            contentRect: NSRect(x: 0, y: 0, width: 390, height: 510),
            styleMask: [.titled, .closable, .utilityWindow],
            backing: .buffered,
            defer: false)
        panel.title = "Quick Terminal Setup"
        panel.level = .floating
        panel.hidesOnDeactivate = false
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
        panel.isReleasedWhenClosed = false
        panel.contentView = NSHostingView(rootView: QuickTerminalSetupView(appDelegate: appDelegate))

        super.init(window: panel)
        panel.delegate = self
        panel.center()
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func showWindow(_ sender: Any?) {
        appDelegate?.refreshQuickTerminalSetupState()
        super.showWindow(sender)
        window?.makeKeyAndOrderFront(sender)

        refreshTimer?.invalidate()
        refreshTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.appDelegate?.refreshQuickTerminalSetupState()
            }
        }
    }

    func windowWillClose(_ notification: Notification) {
        refreshTimer?.invalidate()
        refreshTimer = nil
    }
}

private final class QuickTerminalSetupPanel: NSPanel {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }
}

private struct QuickTerminalSetupView: View {
    @ObservedObject var appDelegate: AppDelegate

    private var isReady: Bool {
        appDelegate.quickTerminalGlobalBindingConfigured &&
            appDelegate.quickTerminalAccessibilityGranted &&
            appDelegate.quickTerminalListenerReady
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(spacing: 14) {
                DraggableAppBundleIcon()
                    .frame(width: 56, height: 56)
                    .accessibilityLabel("Ghostty app icon. Drag to Accessibility settings")
                VStack(alignment: .leading, spacing: 4) {
                    Text("Quick Terminal")
                        .font(.title2.weight(.semibold))
                    Text("Set up a keyboard shortcut that works anywhere.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }

            Divider()

            VStack(alignment: .leading, spacing: 8) {
                Text("GLOBAL SHORTCUT")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                if let shortcut = appDelegate.quickTerminalGlobalShortcut {
                    Text(shortcut)
                        .font(.system(size: 24, weight: .medium, design: .rounded))
                        .accessibilityLabel("Global Quick Terminal shortcut: \(shortcut)")
                } else if appDelegate.quickTerminalGlobalBindingConfigured {
                    Text("Configured shortcut")
                        .font(.headline)
                    Text("This key cannot be shown as a macOS shortcut.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    Text("No global shortcut assigned")
                        .font(.headline)
                    Text("In Settings → Keybind, add `global:ctrl+space` and choose `toggle_quick_terminal`.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                    Button("Open Settings…") {
                        appDelegate.openSettingsEditor(nil)
                    }
                }
            }

            VStack(alignment: .leading, spacing: 10) {
                Text("ACCESSIBILITY")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                readinessRow(
                    title: "Accessibility permission",
                    ready: appDelegate.quickTerminalAccessibilityGranted)
                readinessRow(
                    title: "Global shortcut listener",
                    ready: appDelegate.quickTerminalListenerReady)

                if !appDelegate.quickTerminalAccessibilityGranted {
                    Text("Drag the Ghostty icon above into the Accessibility list, then enable it.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                    Button("Open Accessibility Settings") {
                        openAccessibilitySettings()
                    }
                } else if !appDelegate.quickTerminalListenerReady {
                    Text("Ghostty has permission. The global listener is starting…")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Spacer(minLength: 0)

            HStack {
                if isReady {
                    Label("Ready", systemImage: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                        .accessibilityIdentifier("quickTerminalSetupReady")
                } else {
                    Label("Setup needed", systemImage: "info.circle")
                        .foregroundStyle(.secondary)
                        .accessibilityIdentifier("quickTerminalSetupNotReady")
                }
                Spacer()
                Button("Test Quick Terminal") {
                    appDelegate.testQuickTerminalSetup(nil)
                }
                .keyboardShortcut(.defaultAction)
                .disabled(!isReady)
                .accessibilityIdentifier("testQuickTerminalButton")
            }
        }
        .padding(22)
        .frame(width: 390, height: 510)
    }

    private func readinessRow(title: String, ready: Bool) -> some View {
        HStack(spacing: 8) {
            Image(systemName: ready ? "checkmark.circle.fill" : "circle")
                .foregroundStyle(ready ? Color.green : Color.secondary)
            Text(title)
            Spacer()
            Text(ready ? "Ready" : "Needed")
                .foregroundStyle(.secondary)
        }
        .font(.callout)
        .accessibilityElement(children: .combine)
    }

    private func openAccessibilitySettings() {
        guard let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility") else {
            return
        }
        NSWorkspace.shared.open(url)
    }
}

/// Starts a standard file URL drag so the user can add this app to Accessibility.
private struct DraggableAppBundleIcon: NSViewRepresentable {
    func makeNSView(context: Context) -> DraggableBundleIconView {
        DraggableBundleIconView()
    }

    func updateNSView(_ nsView: DraggableBundleIconView, context: Context) {
        nsView.image = NSWorkspace.shared.icon(forFile: Bundle.main.bundleURL.path)
        nsView.toolTip = "Drag Ghostty to the Accessibility list in System Settings"
    }
}

private final class DraggableBundleIconView: NSImageView, NSDraggingSource {
    override func mouseDragged(with event: NSEvent) {
        let bundleURL = Bundle.main.bundleURL
        let draggingItem = NSDraggingItem(pasteboardWriter: bundleURL as NSURL)
        draggingItem.setDraggingFrame(bounds, contents: image)
        beginDraggingSession(with: [draggingItem], event: event, source: self)
    }

    func draggingSession(
        _ session: NSDraggingSession,
        sourceOperationMaskFor context: NSDraggingContext
    ) -> NSDragOperation {
        .copy
    }

    func ignoreModifierKeys(for session: NSDraggingSession) -> Bool { true }
}
