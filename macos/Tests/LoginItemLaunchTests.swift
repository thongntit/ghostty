import AppKit
import ApplicationServices
import Testing
@testable import Ghostty

@Suite
struct LoginItemLaunchTests {
    @Test func recognizesLoginItemOpenEvent() {
        let event = openEvent(AEEventID(kAEOpenApplication))
        markAsLoginItem(event)

        #expect(AppDelegate.isLoginItemOpenEvent(event))
    }

    @Test func normalOpenAndReopenAreNotLoginLaunches() {
        #expect(!AppDelegate.isLoginItemOpenEvent(nil))
        #expect(!AppDelegate.isLoginItemOpenEvent(openEvent(AEEventID(kAEOpenApplication))))

        let reopen = openEvent(AEEventID(kAEReopenApplication))
        markAsLoginItem(reopen)
        #expect(!AppDelegate.isLoginItemOpenEvent(reopen))
    }

    private func openEvent(_ id: AEEventID) -> NSAppleEventDescriptor {
        NSAppleEventDescriptor(
            eventClass: AEEventClass(kCoreEventClass),
            eventID: id,
            targetDescriptor: nil,
            returnID: AEReturnID(kAutoGenerateReturnID),
            transactionID: AETransactionID(kAnyTransactionID)
        )
    }

    private func markAsLoginItem(_ event: NSAppleEventDescriptor) {
        event.setParam(
            NSAppleEventDescriptor(enumCode: OSType(keyAELaunchedAsLogInItem)),
            forKeyword: AEKeyword(keyAEPropData)
        )
    }
}
