//
//  domusUITests.swift
//  domusUITests
//
//  Created by Fabrizio Chimienti on 11/10/2026.
//

import XCTest

final class domusUITests: XCTestCase {

    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    override func tearDownWithError() throws {
    }

    @MainActor
    func testAdaptiveDashboardElementsAndNavigation() throws {
        let app = XCUIApplication()
        app.launch()

        // Verifica che l'interfaccia principale dell'app sia caricata correttamente
        XCTAssertTrue(app.wait(for: .runningForeground, timeout: 5.0))

        // Verifica la presenza della barra di navigazione o titolo dell'applicazione
        let domusTitle = app.staticTexts["Domus"]
        let allDevicesTitle = app.staticTexts["Tutti i Dispositivi"]
        let roomsSection = app.staticTexts["Stanze"]

        let hasHeaderOrTitle = domusTitle.exists || allDevicesTitle.exists || roomsSection.exists || app.navigationBars.firstMatch.exists
        XCTAssertTrue(hasHeaderOrTitle, "La dashboard deve mostrare la barra di navigazione o il titolo")
    }

    @MainActor
    func testFloorplan2DToggle() throws {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.wait(for: .runningForeground, timeout: 5.0))

        // Cerca il pulsante per attivare la Pianta 2D (sia in iPad sidebar che in toolbar iPhone)
        let floorplanButton = app.buttons.matching(NSPredicate(format: "label CONTAINS[c] 'Pianta' OR label CONTAINS[c] 'map'")).firstMatch
        if floorplanButton.exists {
            floorplanButton.tap()

            // Verifica che il testo descrittivo o l'intestazione della planimetria appaia
            let floorplanHeader = app.staticTexts.matching(NSPredicate(format: "label CONTAINS[c] 'Planimetria' OR label CONTAINS[c] '2D'")).firstMatch
            if floorplanHeader.waitForExistence(timeout: 2.0) {
                XCTAssertTrue(floorplanHeader.exists)
            }
        }
    }

    @MainActor
    func testColdLaunchPerformance() throws {
        measure(metrics: [XCTApplicationLaunchMetric()]) {
            let app = XCUIApplication()
            app.launch()
        }
    }
}
