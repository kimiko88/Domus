// domus-vapid creates a VAPID key pair and stores it in Domus' systemd
// EnvironmentFile without printing the private key to the terminal.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	appnotification "github.com/domus-hub/domus/internal/application/notification"
	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	output := flag.String("output", "/etc/domus/domus.env", "systemd environment file used by Domus")
	subject := flag.String("subject", "", "VAPID contact (mailto:address or https:// URL)")
	flag.Parse()

	vapidSubject := strings.TrimSpace(*subject)
	if vapidSubject == "" {
		fmt.Fprint(os.Stderr, "Contatto VAPID (mailto:admin@example.org oppure https://dominio): ")
		value, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			fatal("impossibile leggere il contatto VAPID: %v", err)
		}
		vapidSubject = strings.TrimSpace(value)
	}
	if err := appnotification.ValidateVAPIDSubject(vapidSubject); err != nil {
		fatal("contatto VAPID non valido: %v", err)
	}

	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fatal("generazione delle chiavi VAPID non riuscita: %v", err)
	}
	if err := appendVAPIDEnvironment(*output, publicKey, privateKey, vapidSubject); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Coppia VAPID creata e salvata in %s\n", *output)
	fmt.Printf("Chiave pubblica: %s\n", publicKey)
	fmt.Println("Chiave privata salvata nel file protetto; non è stata stampata.")
	fmt.Println("Riavvia Domus per caricare le nuove variabili d'ambiente.")
}

func appendVAPIDEnvironment(path, publicKey, privateKey, subject string) error {
	if path == "" {
		return fmt.Errorf("percorso file di configurazione vuoto")
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("il file %s è un collegamento simbolico: rifiuto di modificarlo", path)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("verifica file %s: %w", path, err)
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("lettura %s: %w", path, err)
	}
	for _, line := range strings.Split(string(existing), "\n") {
		key := strings.TrimSpace(strings.SplitN(line, "=", 2)[0])
		switch key {
		case "DOMUS_VAPID_PUBLIC_KEY", "DOMUS_VAPID_PRIVATE_KEY", "DOMUS_VAPID_SUBJECT":
			return fmt.Errorf("%s contiene già variabili VAPID: rimuovile manualmente prima di generare una nuova coppia", path)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creazione directory di configurazione: %w", err)
	}
	content := strings.TrimRight(string(existing), "\n")
	if content != "" {
		content += "\n"
	}
	content += "\n# Domus browser Web Push (private key: keep this file secret)\n"
	content += "DOMUS_VAPID_PUBLIC_KEY=" + publicKey + "\n"
	content += "DOMUS_VAPID_PRIVATE_KEY=" + privateKey + "\n"
	content += "DOMUS_VAPID_SUBJECT=" + subject + "\n"

	temp, err := os.CreateTemp(filepath.Dir(path), ".domus-env-*")
	if err != nil {
		return fmt.Errorf("creazione file temporaneo: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("protezione permessi del file temporaneo: %w", err)
	}
	if _, err := temp.WriteString(content); err != nil {
		_ = temp.Close()
		return fmt.Errorf("scrittura file ambiente: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sincronizzazione file ambiente: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("chiusura file ambiente: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("installazione file ambiente: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("impostazione permessi 0600 su %s: %w", path, err)
	}
	return nil
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "domus-vapid: "+format+"\n", args...)
	os.Exit(1)
}
