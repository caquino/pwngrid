package api

import (
	"encoding/json"
	"errors"
	"github.com/evilsocket/islazy/log"
	"github.com/evilsocket/pwngrid/models"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	ErrEmpty = errors.New("")
)

func (api *API) UnitEnroll(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	clientAddress := strings.Split(r.RemoteAddr, ":")[0]

	var enroll UnitEnrollmentRequest
	if err = json.Unmarshal(body, &enroll); err != nil {
		log.Warning("error while reading enrollment request from %s: %v", clientAddress, err)
		ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	if err = enroll.Validate(); err != nil {
		log.Warning("error while validating enrollment request from %s: %v", clientAddress, err)
		ERROR(w, http.StatusUnprocessableEntity, err)
	}

	err, unit := models.FindUnitByIdentity(api.DB, enroll.Identity)
	if err != nil {
		log.Error("error while searching unit %s: %v", enroll.Identity, err)
		ERROR(w, http.StatusInternalServerError, ErrEmpty)
	}

	if unit == nil {
		log.Info("enrolling new unit for %s: %s", clientAddress, enroll.Identity)

		unit = &models.Unit{
			Address:   clientAddress,
			Identity:  enroll.Identity,
			PublicKey: enroll.PublicKey,
			Token:     "",
			CreatedAt: time.Now(),
		}

		if err = api.DB.Model(&models.Unit{}).Create(unit).Error; err != nil {
			log.Error("error enrolling %s: %v", unit.Identity, err)
			ERROR(w, http.StatusInternalServerError, ErrEmpty)
		}
	}

	unit.Address = clientAddress
	if unit.Token, err = CreateTokenFor(unit); err != nil {
		log.Error("error creating token for %s: %v", unit.Identity, err)
		ERROR(w, http.StatusInternalServerError, ErrEmpty)
	}

	err = api.DB.Model(unit).UpdateColumns(map[string]interface{}{
		"token":      unit.Token,
		"address":    unit.Address,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		log.Error("error setting token for %s: %v", unit.Identity, err)
		ERROR(w, http.StatusInternalServerError, ErrEmpty)
	}

	log.Info("unit %s enrolled: id:%d address:%s token:%s", unit.Identity, unit.ID, unit.Address, unit.Token)

	JSON(w, http.StatusOK, map[string]string{
		"token": unit.Token,
	})
}