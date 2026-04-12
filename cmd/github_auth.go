package cmd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v69/github"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"scuffinger/internal/auth"
	"scuffinger/internal/i18n"
	"scuffinger/internal/vault"
)

func newGitHubAuthCmd() *cobra.Command {
	return &cobra.Command{
		Use:         "auth",
		Short:       i18n.Get(i18n.CmdGitHubAuthShort),
		Long:        i18n.Get(i18n.CmdGitHubAuthLong),
		Annotations: map[string]string{"skipBootstrap": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig()

			clientID := cfg.GitHub.FirstClientID()
			if clientID == "" {
				return fmt.Errorf("%s", i18n.Get(i18n.ErrAuthNoClientID))
			}

			// ── 1. Request device code ───────────────────────────────
			scopes := []string{"repo", "read:org", "workflow"}
			dcr, err := auth.RequestDeviceCode(clientID, scopes)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.Get(i18n.ErrAuthDeviceCode), err)
			}

			// ── 2. Prompt user ───────────────────────────────────────
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", i18n.Get(i18n.MsgAuthDevicePrompt))
			fmt.Fprintf(cmd.OutOrStdout(), "  URL:   %s\n", dcr.VerificationURI)
			fmt.Fprintf(cmd.OutOrStdout(), "  Code:  %s\n\n", dcr.UserCode)
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", i18n.Get(i18n.MsgAuthPolling))

			// ── 3. Poll for token ────────────────────────────────────
			atr, err := auth.PollForToken(clientID, dcr.DeviceCode, dcr.Interval, dcr.ExpiresIn)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.Get(i18n.ErrAuthPoll), err)
			}

			// ── 4. Verify the token works ────────────────────────────
			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: atr.AccessToken})
			ghClient := github.NewClient(oauth2.NewClient(context.Background(), ts))

			user, _, err := ghClient.Users.Get(context.Background(), "")
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.Get(i18n.ErrAuthVerifyToken), err)
			}

			// ── 5. Store securely ────────────────────────────────────
			store := vault.New()

			if err := auth.SaveToken(store, atr.AccessToken); err != nil {
				return fmt.Errorf("%s: %w", i18n.Get(i18n.ErrAuthSaveToken), err)
			}
			if user.Login != nil {
				_ = auth.SaveUser(store, user.GetLogin())
			}

			fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s as %s\n\n", i18n.Get(i18n.MsgAuthSuccess), user.GetLogin())

			// Show token scopes if available
			if atr.Scope != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Scopes: %s\n\n", atr.Scope)
			}

			return nil
		},
	}
}

// RunInteractiveAuth runs the device flow non-interactively (for embedding into other commands).
// Returns the access token on success.
func RunInteractiveAuth(clientID string, w func(string)) (string, error) {
	scopes := []string{"repo", "read:org", "workflow"}

	dcr, err := auth.RequestDeviceCode(clientID, scopes)
	if err != nil {
		return "", err
	}

	w(fmt.Sprintf("\n  %s\n\n  URL:   %s\n  Code:  %s\n\n  %s\n",
		i18n.Get(i18n.MsgAuthDevicePrompt),
		dcr.VerificationURI,
		dcr.UserCode,
		i18n.Get(i18n.MsgAuthPolling),
	))

	atr, err := auth.PollForToken(clientID, dcr.DeviceCode, dcr.Interval, dcr.ExpiresIn)
	if err != nil {
		return "", err
	}

	// Verify
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: atr.AccessToken})
	ghClient := github.NewClient(&http.Client{Transport: &oauth2.Transport{Source: ts}})
	user, _, err := ghClient.Users.Get(context.Background(), "")
	if err != nil {
		return "", err
	}

	store := vault.New()
	_ = auth.SaveToken(store, atr.AccessToken)
	if user.Login != nil {
		_ = auth.SaveUser(store, user.GetLogin())
	}

	w(fmt.Sprintf("  ✓ %s as %s\n", i18n.Get(i18n.MsgAuthSuccess), user.GetLogin()))
	return atr.AccessToken, nil
}
