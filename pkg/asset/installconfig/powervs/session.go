package powervs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"time"

	"github.com/pkg/errors"

	"github.com/IBM-Cloud/power-go-client/ibmpisession"
	"github.com/sirupsen/logrus"

	types "github.com/openshift/installer/pkg/types/powervs"
)

var (
	//reqAuthEnvs = []string{"IBMID", "IBMID_PASSWORD"}
	//optAuthEnvs = []string{"IBMCLOUD_REGION", "IBMCLOUD_ZONE"}
	//debug = false
	defSessionTimeout time.Duration = 9000000000000000000.0
	defRegion                       = "us_south"
)

// Session is an object representing a session for the IBM Power VS API.
type Session struct {
	session *ibmpisession.IBMPISession
}

// GetSession returns an IBM Cloud session by using credentials found in default locations in order:
// env IBMID & env IBMID_PASSWORD,
// ~/.bluemix/config.json ? (see TODO below)
// and, if no creds are found, asks for them

/* @TODO: if you do an `ibmcloud login` (or in my case ibmcloud login --sso), you get
//  a very nice creds file at ~/.bluemix/config.json, with an IAMToken. There's no username,
//  though (just the account's owner id, but that's not the same). It may be necessary
//  to use the IAMToken vs the password env var mentioned here:
//  https://github.com/IBM-Cloud/power-go-client#ibm-cloud-sdk-for-power-cloud

//  Yes, I think we'll need to use the IAMToken. There's a two-factor auth built into the ibmcloud login,
//  so the password alone isn't enough. The IAMToken is generated as a result. So either:
     1) require the user has done this already and pull from the file
     2) ask the user to paste in their IAMToken.
     3) let the password env var be the IAMToken? (Going with this atm since it's how I started)
     4) put it into Platform {userid: , iamtoken: , ...}
*/
func GetSession(ic *types.Platform) (*Session, error) {
	s, err := getPISession(ic)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load credentials")
	}

	return &Session{session: s}, nil
}

/*
//  https://github.com/IBM-Cloud/power-go-client/blob/master/ibmpisession/ibmpowersession.go
*/
func getPISession(ic *types.Platform) (*ibmpisession.IBMPISession, error) {
	var (
		id, key, region, zone string
		err                   error
	)

	id = ic.UserID
	logrus.Debugf("[INFO] IBM Cloud User ID from install-config.yaml: %s", id)
	if len(id) == 0 {
		if id = os.Getenv("IBMID"); len(id) != 0 {
			logrus.Infof("[INFO] IBM Cloud User ID from $IBMID: %s", id)
		} else {
			return nil, fmt.Errorf("No user id could be found")
		}
	}
	if key, err = authFromJson(); err != nil || len(key) == 0 {
		if key = os.Getenv("IBMID_PASSWORD"); len(key) == 0 {
			return nil, errors.New("unable to find login credentials")
		}
	}

	region = ic.Region
	if len(region) == 0 {
		if region, _ = regionFromJson(); len(region) == 0 {
			region = os.Getenv("IBMCLOUD_REGION")
			if r2 := os.Getenv("IC_REGION"); len(r2) > 0 {
				if len(region) > 0 && region != r2 {
					return nil, errors.New(fmt.Sprintf("conflicting values for IBM Cloud Region: IBMCLOUD_REGION: %s and IC_REGION: %s", region, r2))
				}
				if len(region) == 0 {
					region = r2
				}
			}
		}
	}
	if len(region) == 0 {
		logrus.Infof("No region specified. Using default region")
	} else {
		logrus.Debugf("[DEBUG] Using region %s", region)
	}

	zone = ic.Zone
	if len(zone) == 0 {
		// @TODO: query if region is multi-zone? or just pass through err...
		if zone = os.Getenv("IBMCLOUD_ZONE"); len(zone) == 0 {
			zone = region
			logrus.Infof("No zone specified. Using region")
		}
	}

	// @TODO: pass through debug?
	return ibmpisession.New(key, region, false, defSessionTimeout, id, zone)
}

func authFromJson() (string, error) {

	// Doing this the ugly way because I can't find the apikey struct with json definitions to unmarshal onto
	var (
		apiKey string
		apiMap map[string]json.RawMessage
	)
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	fileBuf, err := ioutil.ReadFile(fmt.Sprintf("%s%c.bluemix%capikey.json", user.HomeDir, os.PathSeparator, os.PathSeparator))
	if err != nil {
		logrus.Debugf("[DEBUG] Error reading ~/.bluemix/apikey.json: %s", err)
		return "", err
	}
	if err = json.Unmarshal(fileBuf, &apiMap); err != nil {
		logrus.Debugf("[DEBUG] Error unmarshaling ~/.bluemix/apikey.json: %s", err)
		return "", err
	}
	if err = json.Unmarshal(apiMap["apikey"], &apiKey); err != nil {
		logrus.Debugf("[DEBUG] Error unmarshaling api key from ~/.bluemix/apikey.json: %s", err)
		return "", err
	}
	return apiKey, nil
}

func regionFromJson() (string, error) {
	var (
		region    string
		regionMap map[string]json.RawMessage
	)
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	fileBuf, err := ioutil.ReadFile(fmt.Sprintf("%s%c.bluemix%cconfig.json", user.HomeDir, os.PathSeparator, os.PathSeparator))
	if err != nil {
		logrus.Debugf("[DEBUG] Error reading ~/.bluemix/config.json: %s", err)
		return "", err
	}
	if err = json.Unmarshal(fileBuf, &regionMap); err != nil {
		logrus.Debugf("[DEBUG] Error unmarshaling ~/.bluemix/config.json: %s", err)
		return "", err
	}
	if err = json.Unmarshal(regionMap["region"], &region); err != nil {
		logrus.Debugf("[DEBUG] Error unmarshaling region from ~/.bluemix/config.json: %s", err)
		return "", err
	}
	return region, nil

}
