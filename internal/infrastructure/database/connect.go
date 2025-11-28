package database

import (
	"fmt"
	"time"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/zlog"
)

func ConnectWithRetries(masterDSN string, slaves []string, opts *dbpg.Options, retries int, delaySec int) (*dbpg.DB, error) {
	if retries <= 0 {
		retries = 1
	}
	if delaySec <= 0 {
		delaySec = 1
	}

	var database *dbpg.DB
	var err error

	for i := 0; i < retries; i++ {
		zlog.Logger.Info().Msgf("Database connection attempt %d/%d", i+1, retries)

		database, err = dbpg.New(masterDSN, slaves, opts)
		if err != nil {
			zlog.Logger.Warn().Err(err).Msgf("dbpg.New failed on attempt %d/%d", i+1, retries)
			database = nil
		} else if database.Master == nil {
			err = fmt.Errorf("database.Master is nil")
			zlog.Logger.Warn().Err(err).Msgf("nil master connection on attempt %d/%d", i+1, retries)
		} else if pingErr := database.Master.Ping(); pingErr != nil {
			err = pingErr
			zlog.Logger.Warn().Err(pingErr).Msgf("db ping failed on attempt %d/%d", i+1, retries)
			database.Master.Close()
			for _, s := range database.Slaves {
				if s != nil {
					s.Close()
				}
			}
			database = nil
		} else {
			zlog.Logger.Info().Msg("Database connection established successfully")
			break
		}

		if i < retries-1 {
			time.Sleep(time.Duration(delaySec) * time.Second)
		}
	}

	if err != nil || database == nil {
		return nil, fmt.Errorf("failed to connect to database after %d retries: %w", retries, err)
	}

	return database, nil
}
