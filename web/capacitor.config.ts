export interface CapacitorConfig {
  appId: string
  appName: string
  webDir: string
  server?: {
    androidScheme?: string
    cleartext?: boolean
    url?: string
    [key: string]: any
  }
  plugins?: Record<string, any>
}

const config: CapacitorConfig = {
  appId: 'io.domus.hub',
  appName: 'Domus',
  webDir: 'dist',
  server: {
    androidScheme: 'https',
    cleartext: true
  },
  plugins: {
    SplashScreen: {
      launchShowDuration: 1500,
      backgroundColor: '#0a0f1d',
      showSpinner: false
    },
    PushNotifications: {
      presentationOptions: ['badge', 'sound', 'alert']
    },
    LocalNotifications: {
      smallIcon: 'ic_stat_icon_config_sample',
      iconColor: '#3b82f6',
      sound: 'beep.wav'
    }
  }
}

export default config
