package wallet

import (
	"crypto/rand"
	"os"
	"path/filepath"

	"github.com/rivine/rivine/crypto"
	"github.com/rivine/rivine/modules"
	"github.com/rivine/rivine/persist"
)

const (
	logFile            = modules.WalletDir + ".log"
	settingsFileSuffix = ".json"
	settingsFile       = modules.WalletDir + settingsFileSuffix

	encryptionVerificationLen = 32
)

var (
	settingsMetadata = persist.Metadata{
		Header:  "Wallet Settings",
		Version: "0.4.0",
	}
	seedMetadata = persist.Metadata{
		Header:  "Wallet Seed",
		Version: "0.4.0",
	}
)

// SpendableKeyFile stores an encrypted spendable key on disk.
type SpendableKeyFile struct {
	UID                    UniqueID
	EncryptionVerification crypto.Ciphertext
	SpendableKey           crypto.Ciphertext
}

// WalletPersist contains all data that persists on disk during wallet
// operation.
type WalletPersist struct {
	// EncryptionVerification is an encrypted string that, when decrypted, is
	// 32 '0' bytes. The UID is used to prevent leaking information in the
	// event that the same key gets used for multiple wallets.
	UID                    UniqueID
	EncryptionVerification crypto.Ciphertext

	// The primary seed is used to generate new addresses as they are required.
	// All addresses are tracked and spendable. Only modules.PublicKeysPerSeed
	// keys/addresses can be created per seed, after which a new seed will need
	// to be generated.
	PrimarySeedFile     SeedFile
	PrimarySeedProgress uint64

	// AuxiliarySeedFiles is a set of seeds that the wallet can spend from, but is
	// no longer using to generate addresses. The primary use case is loading
	// backups in the event of lost files or coins. All auxiliary seeds are
	// encrypted using the primary seed encryption password.
	AuxiliarySeedFiles []SeedFile

	// UnseededKeys are list of spendable keys that were not generated by a
	// random seed.
	UnseededKeys []SpendableKeyFile
}

// loadSettings reads the wallet's settings from the wallet's settings file,
// overwriting the settings object in memory. loadSettings should only be
// called at startup.
func (w *Wallet) loadSettings() error {
	return persist.LoadFile(settingsMetadata, &w.persist, filepath.Join(w.persistDir, settingsFile))
}

// saveSettings writes the wallet's settings to the wallet's settings file,
// replacing the existing file.
func (w *Wallet) saveSettings() error {
	return persist.SaveFile(settingsMetadata, w.persist, filepath.Join(w.persistDir, settingsFile))
}

// saveSettingsSync writes the wallet's settings to the wallet's settings file,
// replacing the existing file, and then syncs to disk.
func (w *Wallet) saveSettingsSync() error {
	return persist.SaveFileSync(settingsMetadata, w.persist, filepath.Join(w.persistDir, settingsFile))
}

// initSettings creates the settings object at startup. If a settings file
// exists, the settings file will be loaded into memory. If the settings file
// does not exist, a new.persist file will be created.
func (w *Wallet) initSettings() error {
	// Check if the settings file exists, if not create it.
	settingsFilename := filepath.Join(w.persistDir, settingsFile)
	_, err := os.Stat(settingsFilename)
	if os.IsNotExist(err) {
		_, err = rand.Read(w.persist.UID[:])
		if err != nil {
			return err
		}
		return w.saveSettings()
	} else if err != nil {
		return err
	}

	// Load the settings file if it does exist.
	return w.loadSettings()
}

// initPersist loads all of the wallet's persistence files into memory,
// creating them if they do not exist.
func (w *Wallet) initPersist() error {
	// Create a directory for the wallet without overwriting an existing
	// directory.
	err := os.MkdirAll(w.persistDir, 0700)
	if err != nil {
		return err
	}

	// Start logging.
	w.log, err = persist.NewFileLogger(filepath.Join(w.persistDir, logFile))
	if err != nil {
		return err
	}

	// Load the settings file.
	err = w.initSettings()
	if err != nil {
		return err
	}
	return nil
}

// createBackup creates a backup file at the desired filepath.
func (w *Wallet) createBackup(backupFilepath string) error {
	return persist.SaveFileSync(settingsMetadata, w.persist, backupFilepath)
}

// CreateBackup creates a backup file at the desired filepath.
func (w *Wallet) CreateBackup(backupFilepath string) error {
	if err := w.tg.Add(); err != nil {
		return err
	}
	defer w.tg.Done()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.createBackup(backupFilepath)
}

/*
// LoadBackup loads a backup file from the provided filepath. The backup file
// primary seed is loaded as an auxiliary seed.
func (w *Wallet) LoadBackup(masterKey, backupMasterKey crypto.TwofishKey, backupFilepath string) error {
	if err := w.tg.Add(); err != nil {
		return err
	}
	defer w.tg.Done()

	lockID := w.mu.Lock()
	defer w.mu.Unlock(lockID)

	// Load all of the seed files, check for duplicates, re-encrypt them (but
	// keep the UID), and add them to the WalletPersist object)
	var backupPersist WalletPersist
	err := persist.LoadFile(settingsMetadata, &backupPersist, backupFilepath)
	if err != nil {
		return err
	}
	backupSeeds := append(backupPersist.AuxiliarySeedFiles, backupPersist.PrimarySeedFile)
	TODO: more
}
*/
