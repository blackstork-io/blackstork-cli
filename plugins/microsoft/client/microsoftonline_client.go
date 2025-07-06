package client

import (
	"context"
	"fmt"
	"os"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"

	"github.com/blackstork-io/fabric/internal/plugin/dataspec"
)

const (
	authURL = "https://login.microsoftonline.com"
)

type AcquireTokenFn func(ctx context.Context, tenantId, clientId string, cred confidential.Credential, scopes []string) (string, error)

var AcquireAzureToken AcquireTokenFn = func(ctx context.Context, tenantId, clientId string, cred confidential.Credential, scopes []string) (accessToken string, err error) {
	confidentialClient, err := confidential.New(authURL+"/"+tenantId, clientId, cred)
	if err != nil {
		return
	}

	result, err := confidentialClient.AcquireTokenSilent(ctx, scopes)
	if err != nil {
		// cache miss, authenticate with another AcquireToken... method
		result, err = confidentialClient.AcquireTokenByCredential(ctx, scopes)
		if err != nil {
			return
		}
	}
	accessToken = result.AccessToken
	return
}

func AcquireTokenWithCreds(ctx context.Context, tokenFn AcquireTokenFn, cfg *dataspec.Block, scopes []string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("no configuration provided")
	}

	tenantId := cfg.GetAttrVal("tenant_id").AsString()
	clientId := cfg.GetAttrVal("client_id").AsString()
	clientSecretAttr := cfg.GetAttrVal("client_secret")

	if !clientSecretAttr.IsNull() {
		cred, err := confidential.NewCredFromSecret(clientSecretAttr.AsString())
		if err != nil {
			return "", err
		}
		accessToken, err := tokenFn(ctx, tenantId, clientId, cred, scopes)
		if err != nil {
			return "", err
		}
		return accessToken, nil
	}

	// if client_secret is not provided, try to use private_key
	privateKeyFileAttr := cfg.GetAttrVal("private_key_file")
	privateKeyAttr := cfg.GetAttrVal("private_key")

	if privateKeyFileAttr.IsNull() && privateKeyAttr.IsNull() {
		return "", fmt.Errorf("Either `client_secret` or `private_key` / `private_key_file` arguments must be provided")
	}

	var pemData []byte
	var err error
	if !privateKeyAttr.IsNull() {
		pemData = []byte(privateKeyAttr.AsString())
	} else {
		pemData, err = os.ReadFile(privateKeyFileAttr.AsString())
		if err != nil {
			return "", fmt.Errorf("failed to read private key file: %w", err)
		}
	}

	keyPassphrase := ""
	keyPassphraseAttr := cfg.GetAttrVal("key_passphrase")
	if !keyPassphraseAttr.IsNull() {
		keyPassphrase = keyPassphraseAttr.AsString()
	}

	certs, privateKey, err := confidential.CertFromPEM(pemData, keyPassphrase)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}
	cred, err := confidential.NewCredFromCert(certs, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create credentials from cert: %w", err)
	}
	accessToken, err := tokenFn(ctx, tenantId, clientId, cred, scopes)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}
