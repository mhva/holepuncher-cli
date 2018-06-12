package main

import (
	"encoding/json"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

type sessionCache struct {
	InstanceInfo   *tunnelInstance       `json:"instance_info"`
	CreationParams *tunnelCreationParams `json:"creation_params"`
}

func sessionCacheFilename(runtimeDir string) string {
	return path.Join(runtimeDir, "session.json")
}

func restoreSessionCache(runtimeDir string) (*sessionCache, error) {
	filename := sessionCacheFilename(runtimeDir)
	sessionFile, err := os.Open(filename)
	if err != nil {
		log.WithFields(log.Fields{
			"cause":    err,
			"filename": filename,
		}).Error("Error opening file for reading")
		return nil, err
	}
	defer sessionFile.Close()

	result := &sessionCache{}
	decoder := json.NewDecoder(sessionFile)
	err = decoder.Decode(result)
	if err != nil {
		log.WithFields(log.Fields{
			"cause":    err,
			"filename": filename,
		}).Error("Error parsing session cache")
		return nil, err
	}
	return result, nil
}

func saveSessionCache(cache *sessionCache, runtimeDir string) error {
	filename := sessionCacheFilename(runtimeDir)
	sessionFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.WithFields(log.Fields{
			"cause": err,
			"path":  filename,
		}).Error("Error opening file for writing")
		return err
	}
	defer sessionFile.Close()

	encoder := json.NewEncoder(sessionFile)
	encoder.SetIndent("", "\t")
	if err = encoder.Encode(cache); err != nil {
		log.WithField("cause", err).Error("Error saving session cache")
		return err
	}
	return nil
}

func clearSessionCache(runtimeDir string) error {
	filename := sessionCacheFilename(runtimeDir)
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"cause":    err,
			"filename": filename,
		}).Error("Couldn't clear session cache")
		return err
	}
	return nil
}
