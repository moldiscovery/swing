package swing

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Authorize user using saved credentials, if MFA devices are found
// for this user asks to pick device and input token
func Authorize(ctx context.Context, region string) (aws.Config, error) {
	creds, err := readCache()
	if err != nil {
		return aws.Config{}, err
	}

	if creds != nil && creds.Expiration.After(time.Now().UTC()) {
		// Cache hit, user is still authenticated
		return loadConfig(ctx, region, creds.AccessKey, creds.SecretAccessKey, creds.SessionToken)
	}

	// Region set here is overwritten if found in AWS shared config file
	cfg, err := loadConfig(ctx, region, "", "", "")
	if err != nil {
		return aws.Config{}, err
	}

	_, err = cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return aws.Config{}, err
	}
	devices, err := getMFADevices(ctx, cfg)
	if err != nil {
		return aws.Config{}, err
	}

	var device iamtypes.MFADevice

	if len(devices) > 1 {
		// Ask user which device
		device, err = getDevice(devices)
		if err != nil {
			return aws.Config{}, err
		}
	} else if len(devices) == 1 {
		device = devices[0]
	} else {
		// No MFA, already authorized
		return cfg, nil
	}

	// Ask user for token
	token, err := getToken(*device.SerialNumber)
	if err != nil {
		return aws.Config{}, err
	}

	// Get new credentials
	svc := sts.NewFromConfig(cfg)
	newCredentials, err := svc.GetSessionToken(ctx,
		&sts.GetSessionTokenInput{
			SerialNumber: device.SerialNumber,
			TokenCode:    aws.String(token),
		},
	)
	if err != nil {
		return aws.Config{}, err
	}

	saveCache(&cachedCredentials{
		AccessKey:       *newCredentials.Credentials.AccessKeyId,
		SecretAccessKey: *newCredentials.Credentials.SecretAccessKey,
		SessionToken:    *newCredentials.Credentials.SessionToken,
		Expiration:      *newCredentials.Credentials.Expiration,
	})

	return loadConfig(ctx, region,
		*newCredentials.Credentials.AccessKeyId,
		*newCredentials.Credentials.SecretAccessKey,
		*newCredentials.Credentials.SessionToken,
	)
}

// Loads default AWS config, if accessKey is set static credentials
// are used instead of the default credentials chain
func loadConfig(ctx context.Context, region, accessKey, secretAccessKey, sessionToken string) (aws.Config, error) {
	opts := make([]func(*config.LoadOptions) error, 0)
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}
	if accessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretAccessKey, sessionToken),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("Can't load AWS config: %v", err)
	}
	return cfg, nil
}

type cachedCredentials struct {
	AccessKey       string    `json:"accessKey"`
	SecretAccessKey string    `json:"secretAccessKey"`
	SessionToken    string    `json:"sessionToken"`
	Expiration      time.Time `json:"expiration"`
}

func cachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("Can't find config directory: %v", err)
	}

	configDir = filepath.Join(configDir, "Swing")
	err = os.MkdirAll(configDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("Can't create config directory: %v", err)
	}

	return filepath.Join(configDir, "swing.conf"), nil
}

func readCache() (*cachedCredentials, error) {
	configPath, err := cachePath()
	if err != nil {
		return nil, err
	}

	configs, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("Error reading configs: %v", err)
	}

	var c *cachedCredentials
	err = json.Unmarshal(configs, &c)
	if err != nil {
		return nil, fmt.Errorf("Error parsing configs: %v", err)
	}

	return c, nil
}

func saveCache(c *cachedCredentials) error {
	configPath, err := cachePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("Can't convert credentials to json: %v", err)
	}

	// We ignore the error here, it would be obviously better if credentials
	// are saved to avoid asking a new token each time but we'll take the risk
	ioutil.WriteFile(configPath, data, os.ModePerm)

	return nil
}

// Gets list of MFA devices associated with the user's account
func getMFADevices(ctx context.Context, cfg aws.Config) ([]iamtypes.MFADevice, error) {
	svc := iam.NewFromConfig(cfg)

	userOut, err := svc.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		return make([]iamtypes.MFADevice, 0),
			fmt.Errorf("Error requesting user name: %v", err)
	}

	mfaDevicesOut, err := svc.ListMFADevices(ctx, &iam.ListMFADevicesInput{
		UserName: userOut.User.UserName,
	})
	if err != nil {
		return make([]iamtypes.MFADevice, 0),
			fmt.Errorf("Error requesting MFA devices: %v", err)
	}

	return mfaDevicesOut.MFADevices, nil
}

// Keeps asking user for token generated by selected device
func getToken(mfaDevice string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	token := ""
	for {
		if token != "" {
			break
		}

		fmt.Printf("Enter token for device %s: ", mfaDevice)
		var err error = nil
		token, err = reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("Can't read token input: %v", err)
		}
		token = strings.TrimSpace(token)
	}
	return token, nil
}

// Keeps asking user to choose device for authentication
func getDevice(mfaDevices []iamtypes.MFADevice) (iamtypes.MFADevice, error) {
	reader := bufio.NewReader(os.Stdin)
	for i, device := range mfaDevices {
		fmt.Printf("%d) %s\n", i, *device.SerialNumber)
	}
	deviceIndex := -1
	for {
		if deviceIndex >= 0 && deviceIndex < len(mfaDevices) {
			break
		}

		fmt.Printf("Choose MFA device to authenticate [0-%d]: ", len(mfaDevices)-1)
		device, err := reader.ReadString('\n')
		if err != nil {
			return iamtypes.MFADevice{}, fmt.Errorf("Can't read device input: %v", err)
		}
		deviceIndex, err = strconv.Atoi(strings.TrimSpace(device))
		if err != nil {
			return iamtypes.MFADevice{}, fmt.Errorf("Can't parse device input: %v", err)
		}
	}

	return mfaDevices[deviceIndex], nil
}
