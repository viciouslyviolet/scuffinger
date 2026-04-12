package i18n_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"scuffinger/internal/i18n"
)

var _ = Describe("Messages", func() {
	AfterEach(func() {
		// Reset to English after each test
		i18n.Set(i18n.En)
	})

	Describe("Get", func() {
		It("returns the English message for a known key", func() {
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTP server starting"))
		})

		It("falls back to the key name for an unknown key", func() {
			Expect(i18n.Get("totally.unknown.key")).To(Equal("totally.unknown.key"))
		})
	})

	Describe("Set", func() {
		It("switches the active language", func() {
			custom := i18n.Messages{
				i18n.MsgServerStarting: "Servidor HTTP iniciando",
			}
			i18n.Set(custom)

			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("Servidor HTTP iniciando"))
		})

		It("falls back to key for messages missing in the custom set", func() {
			i18n.Set(i18n.Messages{}) // empty
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal(string(i18n.MsgServerStarting)))
		})
	})

	Describe("Err", func() {
		It("wraps an error with a translated prefix", func() {
			cause := errors.New("dial tcp: connection refused")
			err := i18n.Err(i18n.ErrCachePing, cause)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Cache is not reachable: dial tcp: connection refused"))
			Expect(errors.Is(err, cause)).To(BeTrue())
		})
	})

	Describe("English completeness", func() {
		It("should have a non-empty value for every exported key", func() {
			assertComplete(i18n.En, "En")
		})
	})

	Describe("Spanish completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Es, "Es")
		})

		It("should return Spanish when active", func() {
			i18n.Set(i18n.Es)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("Servidor HTTP iniciando"))
		})
	})

	Describe("French completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Fr, "Fr")
		})

		It("should return French when active", func() {
			i18n.Set(i18n.Fr)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("Démarrage du serveur HTTP"))
		})
	})

	Describe("Maltese completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Mt, "Mt")
		})

		It("should return Maltese when active", func() {
			i18n.Set(i18n.Mt)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("Is-server HTTP qed jibda"))
		})
	})

	Describe("Japanese completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Ja, "Ja")
		})

		It("should return Japanese when active", func() {
			i18n.Set(i18n.Ja)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTPサーバーを起動しています"))
		})
	})

	Describe("Chinese completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Zh, "Zh")
		})

		It("should return Chinese when active", func() {
			i18n.Set(i18n.Zh)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTP服务器正在启动"))
		})
	})

	Describe("German completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.De, "De")
		})

		It("should return German when active", func() {
			i18n.Set(i18n.De)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTP-Server wird gestartet"))
		})
	})

	Describe("Norwegian completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.No, "No")
		})

		It("should return Norwegian when active", func() {
			i18n.Set(i18n.No)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTP-server starter"))
		})
	})

	Describe("Finnish completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Fi, "Fi")
		})

		It("should return Finnish when active", func() {
			i18n.Set(i18n.Fi)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTP-palvelin käynnistyy"))
		})
	})

	Describe("Italian completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.It, "It")
		})

		It("should return Italian when active", func() {
			i18n.Set(i18n.It)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("Avvio del server HTTP"))
		})
	})

	Describe("Swedish completeness", func() {
		It("should have a translation for every key in English", func() {
			assertComplete(i18n.Sv, "Sv")
		})

		It("should return Swedish when active", func() {
			i18n.Set(i18n.Sv)
			Expect(i18n.Get(i18n.MsgServerStarting)).To(Equal("HTTP-server startar"))
		})
	})
})

// allKeys is the authoritative list of every message key.
var allKeys = []i18n.Key{
	i18n.MsgConfigLoaded, i18n.ErrConfigLoad,
	i18n.MsgBootstrapConnecting, i18n.MsgBootstrapSelfTests,
	i18n.MsgBootstrapTestsPassed, i18n.MsgBootstrapHealthStart,
	i18n.ErrBootstrapConnect, i18n.ErrBootstrapSelfTests,
	i18n.MsgManagerConnecting, i18n.MsgManagerConnected,
	i18n.MsgManagerSelfTest, i18n.MsgManagerSelfTestPassed,
	i18n.MsgManagerClosing, i18n.WarnManagerHealthFailed,
	i18n.ErrManagerConnect, i18n.ErrManagerSelfTest,
	i18n.ErrManagerClose, i18n.ErrManagerShutdown,
	i18n.MsgCacheSelfTestInit, i18n.MsgCacheSelfTestPassed,
	i18n.ErrCachePing, i18n.ErrCacheSetInit,
	i18n.ErrCacheGetInit, i18n.ErrCacheInitMismatch,
	i18n.MsgDbCreatingTestDb, i18n.MsgDbRunningCrud,
	i18n.MsgDbSelfTestPassed, i18n.WarnDbDropTestFailed,
	i18n.ErrDbConnect, i18n.ErrDbPing,
	i18n.ErrDbCreateTestDb, i18n.ErrDbConnectTestDb,
	i18n.ErrDbCrud, i18n.ErrDbCreateTable,
	i18n.ErrDbInsert, i18n.ErrDbRead,
	i18n.ErrDbReadMismatch, i18n.ErrDbUpdate,
	i18n.ErrDbReadAfterUpdate, i18n.ErrDbUpdateMismatch,
	i18n.MsgServerStarting, i18n.MsgServerShutdown,
	i18n.MsgServerStopped, i18n.MsgServerRoutes,
	i18n.ErrServerListen, i18n.ErrServerShutdown,
	i18n.MsgHealthReady, i18n.MsgHealthNotReady,
	i18n.MsgGhConnecting, i18n.MsgGhConnected,
	i18n.MsgGhAuthToken, i18n.MsgGhAuthApp,
	i18n.MsgGhSelfTestPassed, i18n.MsgGhRateRemaining,
	i18n.WarnGhRateLow, i18n.ErrGhNotConfigured,
	i18n.ErrGhAuth, i18n.ErrGhReadKey,
	i18n.ErrGhParseKey, i18n.ErrGhInstallToken,
	i18n.ErrGhSelfTest, i18n.ErrGhRateLimit,
	i18n.ErrGhFetchUser, i18n.ErrGhFetchOrg,
	i18n.ErrGhFetchRepo, i18n.ErrGhFetchBranches,
	i18n.ErrGhFetchWorkflows, i18n.ErrGhFetchRuns,
	i18n.ErrGhFetchRateLimit, i18n.ErrGhInvalidWorkflow,
	i18n.MsgGhCollectorStarting, i18n.MsgGhCollectorTick,
	i18n.MsgGhCollectorRepo, i18n.MsgGhCollectorStopped,
	i18n.MsgGhCollectorPassed, i18n.MsgGhCollectorAnnotations,
	i18n.ErrGhCollectorFetchRepo, i18n.ErrGhCollectorFetchRuns,
	i18n.ErrGhCollectorFetchJobs, i18n.ErrGhCollectorFetchAnnotations,
	i18n.ErrGhCollectorNoRepos, i18n.ErrGhCollectorParseRepo,
	i18n.MsgAuthDevicePrompt, i18n.MsgAuthPolling,
	i18n.MsgAuthSuccess, i18n.MsgAuthLoggedOut,
	i18n.MsgAuthStatusLoggedIn, i18n.MsgAuthStatusNoToken,
	i18n.MsgAuthTokenFromVault, i18n.ErrAuthNoClientID,
	i18n.ErrAuthDeviceCode, i18n.ErrAuthPoll,
	i18n.ErrAuthSaveToken, i18n.ErrAuthVerifyToken,
	i18n.CmdRootShort, i18n.CmdRootLong,
	i18n.CmdVersionShort, i18n.CmdVersionLong,
	i18n.CmdServeShort, i18n.CmdServeLong,
	i18n.CmdGitHubShort, i18n.CmdGitHubLong,
	i18n.CmdGitHubAuthShort, i18n.CmdGitHubAuthLong,
	i18n.CmdGitHubStatusShort, i18n.CmdGitHubStatusLong,
	i18n.CmdGitHubLogoutShort, i18n.CmdGitHubLogoutLong,
	i18n.CmdGitHubMonitorShort, i18n.CmdGitHubMonitorLong,
	i18n.CmdFlagConfig,
}

func assertComplete(msgs i18n.Messages, lang string) {
	for _, k := range allKeys {
		v, ok := msgs[k]
		ExpectWithOffset(1, ok).To(BeTrue(), "%s: missing key %q", lang, k)
		ExpectWithOffset(1, v).ToNot(BeEmpty(), "%s: empty value for key %q", lang, k)
	}
}
