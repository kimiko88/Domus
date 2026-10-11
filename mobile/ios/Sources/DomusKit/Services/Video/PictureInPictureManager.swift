import Foundation
import SwiftUI
import Combine

#if canImport(AVKit) && canImport(AVFoundation)
import AVKit
import AVFoundation
#endif

/// Gestore Picture-in-Picture (PiP) per lo streaming live delle telecamere di sicurezza e videocitofono.
@MainActor
public final class PictureInPictureManager: ObservableObject {
    public static let shared = PictureInPictureManager()

    @Published public private(set) var isPiPActive: Bool = false
    @Published public private(set) var isPiPSupported: Bool = false
    @Published public private(set) var activeCameraEntityId: String? = nil

    #if canImport(AVKit) && os(iOS)
    private var pipController: AVPictureInPictureController?
    #endif

    public init() {
        checkSupport()
    }

    private func checkSupport() {
        #if canImport(AVKit) && os(iOS)
        self.isPiPSupported = AVPictureInPictureController.isPictureInPictureSupported()
        #else
        self.isPiPSupported = false
        #endif
    }

    /// Attiva la modalità Picture-in-Picture per la telecamera specificata.
    public func startPictureInPicture(cameraEntityId: String) {
        self.activeCameraEntityId = cameraEntityId
        self.isPiPActive = true

        #if canImport(AVKit) && os(iOS)
        if let pipController = pipController, !pipController.isPictureInPictureActive {
            pipController.startPictureInPicture()
        }
        #endif
    }

    /// Ferma la modalità Picture-in-Picture.
    public func stopPictureInPicture() {
        #if canImport(AVKit) && os(iOS)
        if let pipController = pipController, pipController.isPictureInPictureActive {
            pipController.stopPictureInPicture()
        }
        #endif

        self.isPiPActive = false
        self.activeCameraEntityId = nil
    }

    /// Inverte lo stato Picture-in-Picture per una data telecamera.
    public func togglePictureInPicture(cameraEntityId: String) {
        if isPiPActive && activeCameraEntityId == cameraEntityId {
            stopPictureInPicture()
        } else {
            startPictureInPicture(cameraEntityId: cameraEntityId)
        }
    }
}
