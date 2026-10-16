import XCTest
@testable import DomusKit

@MainActor
final class LocalizationTests: XCTestCase {
    func testLanguageDefinitionsAndDisplayNames() {
        XCTAssertEqual(DomusLanguage.allCases.count, 5)
        XCTAssertEqual(DomusLanguage.it.displayName, "Italiano")
        XCTAssertEqual(DomusLanguage.en.displayName, "English")
        XCTAssertEqual(DomusLanguage.fr.displayName, "Français")
        XCTAssertEqual(DomusLanguage.es.displayName, "Español")
        XCTAssertEqual(DomusLanguage.de.displayName, "Deutsch")

        XCTAssertEqual(DomusLanguage.it.flag, "🇮🇹")
        XCTAssertEqual(DomusLanguage.en.flag, "🇬🇧")
        XCTAssertEqual(DomusLanguage.fr.flag, "🇫🇷")
        XCTAssertEqual(DomusLanguage.es.flag, "🇪🇸")
        XCTAssertEqual(DomusLanguage.de.flag, "🇩🇪")
    }

    func testTranslationsAcrossAllFiveLanguages() {
        let manager = DomusLocalizationManager.shared

        // Test Italian (Default)
        manager.currentLanguage = .it
        XCTAssertEqual(manager.tr("common_ok"), "OK")
        XCTAssertEqual(manager.tr("nav_dashboard"), "Dashboard")
        XCTAssertEqual(manager.tr("chores_new_task"), "Nuova Faccenda")

        // Test English
        manager.currentLanguage = .en
        XCTAssertEqual(manager.tr("common_ok"), "OK")
        XCTAssertEqual(manager.tr("chores_new_task"), "New Chore")

        // Test French
        manager.currentLanguage = .fr
        XCTAssertEqual(manager.tr("nav_dashboard"), "Tableau de bord")
        XCTAssertEqual(manager.tr("chores_new_task"), "Nouvelle Tâche")

        // Test Spanish
        manager.currentLanguage = .es
        XCTAssertEqual(manager.tr("nav_dashboard"), "Panel de Control")
        XCTAssertEqual(manager.tr("chores_new_task"), "Nueva Tarea")

        // Test German
        manager.currentLanguage = .de
        XCTAssertEqual(manager.tr("common_ok"), "OK")
        XCTAssertEqual(manager.tr("chores_new_task"), "Neue Aufgabe")

        // Reset to Italian
        manager.currentLanguage = .it
    }
}
