// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "DomusMobile",
    platforms: [
        .iOS(.v17),
        .macOS(.v14)
    ],
    products: [
        .library(
            name: "DomusKit",
            targets: ["DomusKit"]
        )
    ],
    dependencies: [],
    targets: [
        // Libreria condivisa: Domain (Models, UseCases, Repositories) + Data Layer
        .target(
            name: "DomusKit",
            dependencies: [],
            path: "Sources/DomusKit"
            // Il path "Sources/DomusKit" include automaticamente:
            //   Domain/Models/, Domain/UseCases/, Domain/Repositories/
            //   Data/Network/, Data/Repositories/, Data/Storage/
        ),
        .testTarget(
            name: "DomusKitTests",
            dependencies: ["DomusKit"],
            path: "Tests/DomusKitTests"
        )
    ]
)
