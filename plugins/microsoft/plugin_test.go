// Copyright 2026 BlackStork BV
//
// Use of this software is governed by the Business Source License included in the
// file LICENSE and at www.mariadb.com/bsl11.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file .licenses/APACHE-2.0.txt.

package microsoft

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"

	"github.com/blackstork-io/blackstork-cli/plugin/plugintest"
	"github.com/blackstork-io/blackstork-cli/specs/dataspec"
)

func TestPlugin_Schema(t *testing.T) {
	schema := Plugin("1.2.3", nil, nil, nil, nil)
	assert.Equal(t, "blackstork/microsoft", schema.Name)
	assert.Equal(t, "1.2.3", schema.Version)
	assert.NotNil(t, schema.DataSources["microsoft_sentinel_incidents"])
	assert.NotNil(t, schema.DataSources["microsoft_graph"])
	assert.NotNil(t, schema.DataSources["microsoft_security"])
	assert.NotNil(t, schema.ContentProviders["azure_openai_text"])
}

func TestMakeDefaultGraphClientLoader(t *testing.T) {
	loader := MakeDefaultMicrosoftGraphClientLoader(func(ctx context.Context, tenantId, clientId string, cred confidential.Credential, scopes []string) (string, error) {
		return "test-token", nil
	})
	assert.NotNil(t, loader)

	plugin := Plugin("1.0.0", nil, nil, func(ctx context.Context, apiVersion string, cfg *dataspec.Block) (client MicrosoftGraphClient, err error) {
		return nil, nil
	}, nil)
	t.Run("with client secret", func(t *testing.T) {
		result := plugintest.NewTestDecoder(t, plugin.DataSources["microsoft_graph"].Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode()
		client, err := loader(context.Background(), "2023-11-01", result)
		assert.Nil(t, err)
		assert.NotNil(t, client)
	})

	t.Run("should create with client secret", func(t *testing.T) {
		result := plugintest.NewTestDecoder(t, plugin.DataSources["microsoft_graph"].Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("client_secret", cty.StringVal("csecret")).
			Decode()
		client, err := loader(context.Background(), "2023-11-01", result)
		assert.Nil(t, err)
		assert.NotNil(t, client)
	})

	t.Run("should fail if no auth specified", func(t *testing.T) {
		result := plugintest.NewTestDecoder(t, plugin.DataSources["microsoft_graph"].Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			Decode()
		_, err := loader(context.Background(), "2023-11-01", result)
		assert.NotNil(t, err)
		assert.EqualError(t, err, "Either `client_secret` or `private_key` / `private_key_file` arguments must be provided")
	})

	t.Run("should use private key contents if specified", func(t *testing.T) {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		assert.Nil(t, err)
		key, err := x509.MarshalPKCS8PrivateKey(privateKey)
		assert.Nil(t, err)

		block := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: key,
		}
		buff := bytes.NewBuffer([]byte{})
		err = pem.Encode(buff, block)
		assert.Nil(t, err)
		// add certificate
		cert := x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: "test"},
		}
		certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &privateKey.PublicKey, privateKey)
		assert.Nil(t, err)
		certBlock := &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}
		err = pem.Encode(buff, certBlock)
		assert.Nil(t, err)
		result := plugintest.NewTestDecoder(t, plugin.DataSources["microsoft_graph"].Config).
			SetAttr("client_id", cty.StringVal("cid")).
			SetAttr("tenant_id", cty.StringVal("tid")).
			SetAttr("private_key", cty.StringVal(buff.String())).
			Decode()
		client, err := loader(context.Background(), "2023-11-01", result)
		assert.Nil(t, err)
		assert.NotNil(t, client)
	})
}
