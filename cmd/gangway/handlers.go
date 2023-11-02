// Copyright © 2017 Heptio
// Copyright © 2017 Craig Tracey
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jcrood/gangway/internal/config"
	"github.com/jcrood/gangway/templates"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"
)

// userInfo stores information about an authenticated user
type userInfo struct {
	ClusterName  string
	Username     string
	Claims       map[string]interface{}
	KubeCfgUser  string
	IDToken      string
	RefreshToken string
	ClientID     string
	ClientSecret string
	IssuerURL    string
	APIServerURL string
	ClusterCA    string
	TrustedCA    string
	ShowClaims   bool
	HTTPPath     string
}

type clusterHomeInfo struct {
	Clusters map[string][]config.Config
	HTTPPath string
}

// homeInfo is used to store dynamic properties on
type homeInfo struct {
	ClusterName string
	HTTPPath    string
}

func serveTemplate(tmplFile string, data interface{}, w http.ResponseWriter) {
	var (
		templatePath string
		templateData []byte
		err          error
	)

	// Use custom templates if provided
	if clusterCfg.CustomHTMLTemplatesDir != "" {
		templatePath = filepath.Join(clusterCfg.CustomHTMLTemplatesDir, tmplFile)
		templateData, err = os.ReadFile(templatePath)
	} else {
		templateData, err = templates.FS.ReadFile(tmplFile)
	}

	if err != nil {
		log.Errorf("Failed to find template asset: %s at path: %s", tmplFile, templatePath)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := htmltemplate.New(tmplFile).Funcs(FuncMap())
	tmpl, err = tmpl.Parse(string(templateData))
	if err != nil {
		log.Errorf("Failed to parse template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = tmpl.ExecuteTemplate(w, tmplFile, data)
	if err != nil {
		log.Errorf("Failed to render template %s: %s", tmplFile, err)
	}
}

func generateKubeConfig(cfg *userInfo) clientcmdapi.Config {
	// fill out kubeconfig structure
	kcfg := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: cfg.ClusterName,
		Clusters: []clientcmdapi.NamedCluster{
			{
				Name: cfg.ClusterName,
				Cluster: clientcmdapi.Cluster{
					Server:                   cfg.APIServerURL,
					CertificateAuthorityData: []byte(cfg.ClusterCA),
				},
			},
		},
		Contexts: []clientcmdapi.NamedContext{
			{
				Name: cfg.ClusterName,
				Context: clientcmdapi.Context{
					Cluster:  cfg.ClusterName,
					AuthInfo: cfg.KubeCfgUser,
				},
			},
		},
		AuthInfos: []clientcmdapi.NamedAuthInfo{
			{
				Name: cfg.KubeCfgUser,
				AuthInfo: clientcmdapi.AuthInfo{
					AuthProvider: &clientcmdapi.AuthProviderConfig{
						Name: "oidc",
						Config: map[string]string{
							"client-id":                      cfg.ClientID,
							"client-secret":                  cfg.ClientSecret,
							"id-token":                       cfg.IDToken,
							"idp-issuer-url":                 cfg.IssuerURL,
							"idp-certificate-authority-data": base64.StdEncoding.EncodeToString([]byte(cfg.TrustedCA)),
							"refresh-token":                  cfg.RefreshToken,
						},
					},
				},
			},
		},
	}
	return kcfg
}

func loginRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := gangwayUserSession.Session.Get(r, "gangway_id_token")
		if err != nil {
			http.Redirect(w, r, clusterCfg.GetRootPathPrefix(), http.StatusTemporaryRedirect)
			return
		}

		if session.Values["id_token"] == nil {
			http.Redirect(w, r, clusterCfg.GetRootPathPrefix(), http.StatusTemporaryRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clustersHome(w http.ResponseWriter, _ *http.Request) {

	data := &clusterHomeInfo{
		Clusters: clusterCfg.Clusters,
		HTTPPath: clusterCfg.HTTPPath,
	}

	serveTemplate("clustersHome.tmpl", data, w)
}

func homeHandler(w http.ResponseWriter, _ *http.Request) {
	data := &homeInfo{
		ClusterName: cfg.ClusterName,
		HTTPPath:    clusterCfg.HTTPPath,
	}

	serveTemplate("home.tmpl", data, w)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	clusterName := r.URL.Query().Get("cluster")

	if clusterName == "" {
		// Si aucun cluster n'est spécifié, redirigez vers la page de sélection du cluster.
		http.Redirect(w, r, clusterCfg.GetRootPathPrefix(), http.StatusSeeOther)
		return
	}

	clusterConfig, ok := getClusterConfig(clusterName)
	if !ok {
		http.Error(w, "Invalid cluster name", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, clusterConfig.ProviderURL)
	if err != nil {
		log.Errorf("Could not create OIDC provider for cluster %s: %s", clusterName, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	verifier = provider.Verifier(&oidc.Config{ClientID: clusterConfig.ClientID})

	oauth2Cfg = &oauth2.Config{
		ClientID:     clusterConfig.ClientID,
		ClientSecret: clusterConfig.ClientSecret,
		RedirectURL:  clusterConfig.RedirectURL,
		Scopes:       clusterConfig.Scopes,
		Endpoint:     provider.Endpoint(),
	}

	b := make([]byte, 32)
	_, err = rand.Read(b)
	if err != nil {
		log.Errorf("failed to geenrate rnd data: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := base64.URLEncoding.EncodeToString(b)

	session, err := gangwayUserSession.Session.Get(r, "gangway")
	if err != nil {
		log.Errorf("Got an error in login: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["state"] = state
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["clusterName"] = clusterName
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	audience := oauth2.SetAuthURLParam("audience", clusterConfig.Audience)
	offlineAccessType := oauth2.SetAuthURLParam("access_type", "offline")
	forceConsentPrompt := oauth2.SetAuthURLParam("prompt", "consent")
	url := oauth2Cfg.AuthCodeURL(state, audience, offlineAccessType, forceConsentPrompt)
	fmt.Println(url)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	gangwayUserSession.Cleanup(w, r, "gangway")
	gangwayUserSession.Cleanup(w, r, "gangway_id_token")
	gangwayUserSession.Cleanup(w, r, "gangway_refresh_token")
	http.Redirect(w, r, clusterCfg.GetRootPathPrefix(), http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), oauth2.HTTPClient, transportConfig.HTTPClient)

	// load up session cookies
	session, err := gangwayUserSession.Session.Get(r, "gangway")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	clusterName, ok := session.Values["clusterName"].(string)
	if !ok || clusterName == "" {
		http.Error(w, "Internal error: clusterName not found", http.StatusInternalServerError)
		return
	}

	sessionIDToken, err := gangwayUserSession.Session.Get(r, "gangway_id_token")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sessionRefreshToken, err := gangwayUserSession.Session.Get(r, "gangway_refresh_token")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// verify the state string
	state := r.URL.Query().Get("state")
	if state != session.Values["state"] {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	// use the access code to retrieve a token
	code := r.URL.Query().Get("code")
	oauth2Token, err := oauth2Cfg.Exchange(ctx, code)
	// token, err := o2token.Exchange(ctx, code)
	if err != nil {
		log.Errorf("failed to exchange token: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Voici le contenu du oauth2Token : %v", oauth2Token)

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		log.Errorf("no id_token found")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Errorf("failed to verify token: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sessionIDToken.Values["id_token"] = rawIDToken
	sessionRefreshToken.Values["refresh_token"] = oauth2Token.RefreshToken

	// save the session cookies
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = sessionIDToken.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = sessionRefreshToken.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/commandline?cluster=%s", clusterCfg.HTTPPath, clusterName), http.StatusSeeOther)
}

func commandlineHandler(w http.ResponseWriter, r *http.Request) {
	info := generateInfo(w, r)
	if info == nil {
		// generateInfo writes to the ResponseWriter if it encounters an error.
		// TODO(abrand): Refactor this.
		return
	}

	serveTemplate("commandline.tmpl", info, w)
}

func kubeConfigHandler(w http.ResponseWriter, r *http.Request) {
	info := generateInfo(w, r)
	if info == nil {
		// generateInfo writes to the ResponseWriter if it encounters an error.
		// TODO(abrand): Refactor this.
		return
	}

	d, err := yaml.Marshal(generateKubeConfig(info))
	if err != nil {
		log.Errorf("Error creating kubeconfig - %s", err.Error())
		http.Error(w, "Error creating kubeconfig", http.StatusInternalServerError)
		return
	}

	// tell the browser the returned content should be downloaded
	w.Header().Add("Content-Disposition", "Attachment")
	_, err = w.Write(d)
	if err != nil {
		log.Errorf("Failed to write kubeconfig: %v", err)
	}
}

func generateInfo(w http.ResponseWriter, r *http.Request) *userInfo {
	// load the session cookies
	sessionIDToken, err := gangwayUserSession.Session.Get(r, "gangway_id_token")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil
	}
	sessionRefreshToken, err := gangwayUserSession.Session.Get(r, "gangway_refresh_token")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil
	}

	rawIDToken, ok := sessionIDToken.Values["id_token"].(string)
	if !ok {
		gangwayUserSession.Cleanup(w, r, "gangway")
		gangwayUserSession.Cleanup(w, r, "gangway_id_token")
		gangwayUserSession.Cleanup(w, r, "gangway_refresh_token")

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil
	}

	refreshToken, ok := sessionRefreshToken.Values["refresh_token"].(string)
	if !ok {
		gangwayUserSession.Cleanup(w, r, "gangway")
		gangwayUserSession.Cleanup(w, r, "gangway_id_token")
		gangwayUserSession.Cleanup(w, r, "gangway_refresh_token")

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil
	}

	ctx := context.WithValue(r.Context(), oauth2.HTTPClient, transportConfig.HTTPClient)

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Errorf("failed to verify token: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	claims := make(map[string]interface{})
	err = idToken.Claims(&claims)
	if err != nil {
		log.Errorf("failed to unmarshal claims: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil
	}

	clusterName := r.URL.Query().Get("cluster")

	if clusterName == "" {
		// Si aucun cluster n'est spécifié, redirigez vers la page de sélection du cluster.
		http.Redirect(w, r, clusterCfg.GetRootPathPrefix(), http.StatusSeeOther)
		return nil
	}

	cfg, ok := getClusterConfig(clusterName)
	if !ok {
		http.Error(w, "Invalid cluster name", http.StatusBadRequest)
		return nil
	}

	username, ok := claims[cfg.UsernameClaim].(string)
	if !ok {
		http.Error(w, "Could not parse Username claim", http.StatusInternalServerError)
		return nil
	}

	kubeCfgUser := strings.Join([]string{username, cfg.ClusterName}, "@")

	if cfg.EmailClaim != "" {
		log.Warn("using the Email Claim config setting is deprecated. Gangway uses `UsernameClaim@ClusterName`. This field will be removed in a future version.")
	}

	issuerURL, ok := claims["iss"].(string)
	if !ok {
		http.Error(w, "Could not parse Issuer URL claim", http.StatusInternalServerError)
		return nil
	}

	if cfg.ClientSecret == "" {
		log.Warn("Setting an empty Client Secret should only be done if you have no other option and is an inherent security risk.")
	}

	info := &userInfo{
		ClusterName:  cfg.ClusterName,
		Username:     username,
		Claims:       claims,
		KubeCfgUser:  kubeCfgUser,
		IDToken:      rawIDToken,
		RefreshToken: refreshToken,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		IssuerURL:    issuerURL,
		APIServerURL: cfg.APIServerURL,
		ClusterCA:    string(cfg.ClusterCA),
		TrustedCA:    string(clusterCfg.TrustedCA),
		ShowClaims:   cfg.ShowClaims,
		HTTPPath:     clusterCfg.HTTPPath,
	}
	return info
}

func getClusterConfig(clusterName string) (config.Config, bool) {
	for _, clusters := range clusterCfg.Clusters {
		for _, cluster := range clusters {
			if cluster.ClusterName == clusterName {
				return cluster, true
			}
		}
	}
	return config.Config{}, false
}
