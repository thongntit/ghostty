import Foundation
import Testing
@testable import Ghostty

@Suite
struct QuickTerminalTabManagerTests {
    @MainActor
    @Test func createsSwitchesAndClosesTabs() throws {
        let configURL = FileManager.default.temporaryDirectory
            .appendingPathComponent("ghostty-quick-tabs-\(UUID().uuidString).conf")
        try "window-save-state = never\n".write(to: configURL, atomically: true, encoding: .utf8)
        defer { try? FileManager.default.removeItem(at: configURL) }

        let ghostty = Ghostty.App(configPath: configURL.path)
        try #require(ghostty.app != nil)
        let controller = QuickTerminalController(ghostty)
        let manager = controller.tabManager
        #expect(manager.tabs.count == 1)
        let first = try #require(manager.currentTab)

        manager.addNewTab()
        #expect(manager.tabs.count == 2)
        let second = try #require(manager.currentTab)
        #expect(second.id != first.id)

        manager.selectPreviousTab()
        #expect(manager.currentTab?.id == first.id)
        manager.selectNextTab()
        #expect(manager.currentTab?.id == second.id)

        manager.closeTab(second, withConfirmation: false)
        #expect(manager.tabs.count == 1)
        #expect(manager.currentTab?.id == first.id)
    }
}
