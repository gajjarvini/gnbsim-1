// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"
	"gnbsim/common"
	"gnbsim/factory"
	"gnbsim/profile/context"
	"gnbsim/profile/util"
	"gnbsim/simue"
	simuectx "gnbsim/simue/context"
	"strconv"
	"sync"
	"time"
)

//profile names
const (
	REGISTER             string = "register"
	PDU_SESS_EST         string = "pdusessest"
	DEREGISTER           string = "deregister"
	AN_RELEASE           string = "anrelease"
	UE_TRIGG_SERVICE_REQ string = "uetriggservicereq"
)

func InitializeAllProfiles() {
	for _, profile := range factory.AppConfig.Configuration.Profiles {
		profile.Init()
	}
}

func ExecuteProfile(profile *context.Profile, summaryChan chan common.InterfaceMessage) {
	initEventMap(profile)
	initProcedureList(profile)

	gnb, err := factory.AppConfig.Configuration.GetGNodeB(profile.GnbName)
	if err != nil {
		profile.Log.Errorln("GetGNodeB returned:", err)
	}

	imsi, err := strconv.Atoi(profile.StartImsi)
	if err != nil {
		profile.Log.Fatalln("invalid imsi value")
	}
	var wg sync.WaitGroup
	summary := &common.SummaryMessage{
		ProfileType: profile.ProfileType,
		ProfileName: profile.Name,
		ErrorList:   make([]error, 0, 10),
	}

	// Currently executing profile for one IMSI at a time
	for count := 1; count <= profile.UeCount; count++ {
		simUe := simuectx.NewSimUe("imsi-"+strconv.Itoa(imsi), gnb, profile)

		wg.Add(1)
		go simue.Init(simUe, &wg)
		util.SendToSimUe(simUe, common.PROFILE_START_EVENT)

		msg := <-profile.ReadChan
		switch msg.Event {
		case common.PROFILE_PASS_EVENT:
			profile.Log.Infoln("Result: PASS, imsi:", msg.Supi)
			summary.UePassedCount++
		case common.PROFILE_FAIL_EVENT:
			err := fmt.Errorf("imsi:%v, procedure:%v, error:%v", msg.Supi, msg.Proc, msg.Error)
			profile.Log.Infoln("Result: FAIL,", err)
			summary.UeFailedCount++
			summary.ErrorList = append(summary.ErrorList, err)
		}
		time.Sleep(2 * time.Second)
		imsi++
	}

	summaryChan <- summary
	wg.Wait()
}

func initEventMap(profile *context.Profile) {
	switch profile.ProfileType {
	case REGISTER:
		profile.Events = map[common.EventType]common.EventType{
			common.REG_REQUEST_EVENT:     common.AUTH_REQUEST_EVENT,
			common.AUTH_REQUEST_EVENT:    common.AUTH_RESPONSE_EVENT,
			common.SEC_MOD_COMMAND_EVENT: common.SEC_MOD_COMPLETE_EVENT,
			common.REG_ACCEPT_EVENT:      common.REG_COMPLETE_EVENT,
			common.PROFILE_PASS_EVENT:    common.QUIT_EVENT,
		}
	case PDU_SESS_EST:
		profile.Events = map[common.EventType]common.EventType{
			common.REG_REQUEST_EVENT:          common.AUTH_REQUEST_EVENT,
			common.AUTH_REQUEST_EVENT:         common.AUTH_RESPONSE_EVENT,
			common.SEC_MOD_COMMAND_EVENT:      common.SEC_MOD_COMPLETE_EVENT,
			common.REG_ACCEPT_EVENT:           common.REG_COMPLETE_EVENT,
			common.PDU_SESS_EST_REQUEST_EVENT: common.PDU_SESS_EST_ACCEPT_EVENT,
			common.PDU_SESS_EST_ACCEPT_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
			common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
		}
	case DEREGISTER:
		profile.Events = map[common.EventType]common.EventType{
			common.REG_REQUEST_EVENT:           common.AUTH_REQUEST_EVENT,
			common.AUTH_REQUEST_EVENT:          common.AUTH_RESPONSE_EVENT,
			common.SEC_MOD_COMMAND_EVENT:       common.SEC_MOD_COMPLETE_EVENT,
			common.REG_ACCEPT_EVENT:            common.REG_COMPLETE_EVENT,
			common.PDU_SESS_EST_REQUEST_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
			common.PDU_SESS_EST_ACCEPT_EVENT:   common.PDU_SESS_EST_ACCEPT_EVENT,
			common.DEREG_REQUEST_UE_ORIG_EVENT: common.DEREG_ACCEPT_UE_ORIG_EVENT,
			common.PROFILE_PASS_EVENT:          common.QUIT_EVENT,
		}
	case AN_RELEASE:
		profile.Events = map[common.EventType]common.EventType{
			common.REG_REQUEST_EVENT:          common.AUTH_REQUEST_EVENT,
			common.AUTH_REQUEST_EVENT:         common.AUTH_RESPONSE_EVENT,
			common.SEC_MOD_COMMAND_EVENT:      common.SEC_MOD_COMPLETE_EVENT,
			common.REG_ACCEPT_EVENT:           common.REG_COMPLETE_EVENT,
			common.PDU_SESS_EST_REQUEST_EVENT: common.PDU_SESS_EST_ACCEPT_EVENT,
			common.PDU_SESS_EST_ACCEPT_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
			common.TRIGGER_AN_RELEASE_EVENT:   common.CONNECTION_RELEASE_REQUEST_EVENT,
			common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
		}
	case UE_TRIGG_SERVICE_REQ:
		profile.Events = map[common.EventType]common.EventType{
			common.REG_REQUEST_EVENT:          common.AUTH_REQUEST_EVENT,
			common.AUTH_REQUEST_EVENT:         common.AUTH_RESPONSE_EVENT,
			common.SEC_MOD_COMMAND_EVENT:      common.SEC_MOD_COMPLETE_EVENT,
			common.REG_ACCEPT_EVENT:           common.REG_COMPLETE_EVENT,
			common.PDU_SESS_EST_REQUEST_EVENT: common.PDU_SESS_EST_ACCEPT_EVENT,
			common.PDU_SESS_EST_ACCEPT_EVENT:  common.PDU_SESS_EST_ACCEPT_EVENT,
			common.SERVICE_REQUEST_EVENT:      common.SERVICE_ACCEPT_EVENT,
			common.TRIGGER_AN_RELEASE_EVENT:   common.CONNECTION_RELEASE_REQUEST_EVENT,
			common.PROFILE_PASS_EVENT:         common.QUIT_EVENT,
		}
	}
}

func initProcedureList(profile *context.Profile) {
	switch profile.ProfileType {
	case REGISTER:
		profile.Procedures = []common.ProcedureType{common.REGISTRATION_PROCEDURE}
	case PDU_SESS_EST:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
		}
	case DEREGISTER:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.UE_INITIATED_DEREGISTRATION_PROCEDURE,
		}
	case AN_RELEASE:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.AN_RELEASE_PROCEDURE,
		}
	case UE_TRIGG_SERVICE_REQ:
		profile.Procedures = []common.ProcedureType{
			common.REGISTRATION_PROCEDURE,
			common.PDU_SESSION_ESTABLISHMENT_PROCEDURE,
			common.USER_DATA_PKT_GENERATION_PROCEDURE,
			common.AN_RELEASE_PROCEDURE,
			common.UE_TRIGGERED_SERVICE_REQUEST_PROCEDURE,
		}
	}
}
