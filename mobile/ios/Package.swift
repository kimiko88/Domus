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
        ),
        .executable(
            name: "DomusApp",
            targets: ["DomusApp"]
        )
    ],
    dependencies: [],
    targets: [
        .target(
            name: "DomusKit",
            dependencies: [],
            path: "Sources/DomusKit"
        ),
        .executableTarget(
            name: "DomusApp",
            dependencies: ["DomusKit"],
            path: "Sources/DomusApp"
        ),
        .testTarget(
            name: "DomusKitTests",
            dependencies: ["DomusKit"],
            path: "Tests/DomusKitTests"
        )
    ]
)
