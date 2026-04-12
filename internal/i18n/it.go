package i18n

// ── Italiano (Italian) ───────────────────────────────────────────────────────

var It = Messages{
	// Config
	MsgConfigLoaded: "Configurazione caricata",
	ErrConfigLoad:   "Impossibile caricare la configurazione",

	// Bootstrap
	MsgBootstrapConnecting:  "Connessione ai servizi di supporto",
	MsgBootstrapSelfTests:   "Esecuzione degli autotest di avvio",
	MsgBootstrapTestsPassed: "Tutti gli autotest superati",
	MsgBootstrapHealthStart: "Controlli di integrità periodici avviati",
	ErrBootstrapConnect:     "Impossibile connettersi ai servizi richiesti",
	ErrBootstrapSelfTests:   "Autotest di avvio falliti",

	// Manager
	MsgManagerConnecting:     "Connessione al servizio",
	MsgManagerConnected:      "Connesso al servizio",
	MsgManagerSelfTest:       "Esecuzione dell'autotest",
	MsgManagerSelfTestPassed: "Autotest superato",
	MsgManagerClosing:        "Arresto dei servizi",
	WarnManagerHealthFailed:  "Controllo di integrità fallito",
	ErrManagerConnect:        "Connessione al servizio fallita",
	ErrManagerSelfTest:       "Autotest fallito per il servizio",
	ErrManagerClose:          "Errore durante la chiusura del servizio",
	ErrManagerShutdown:       "Si sono verificati errori durante l'arresto",

	// Cache
	MsgCacheSelfTestInit:   "Impostazione della chiave init con il timestamp corrente",
	MsgCacheSelfTestPassed: "Autotest della cache superato",
	ErrCachePing:           "Cache non raggiungibile",
	ErrCacheSetInit:        "Impossibile scrivere la chiave init nella cache",
	ErrCacheGetInit:        "Impossibile leggere la chiave init dalla cache",
	ErrCacheInitMismatch:   "Valore init della cache non corrispondente",

	// Database
	MsgDbCreatingTestDb:  "Creazione del database di test temporaneo",
	MsgDbRunningCrud:     "Esecuzione della verifica CRUD",
	MsgDbSelfTestPassed:  "Autotest del database superato",
	WarnDbDropTestFailed: "Impossibile eliminare il database di test (nuovo tentativo al prossimo avvio)",
	ErrDbConnect:         "Database non raggiungibile",
	ErrDbPing:            "Il database non ha risposto al ping",
	ErrDbCreateTestDb:    "Impossibile creare il database di test",
	ErrDbConnectTestDb:   "Impossibile connettersi al database di test",
	ErrDbCrud:            "Verifica CRUD del database fallita",
	ErrDbCreateTable:     "Impossibile creare la tabella di test",
	ErrDbInsert:          "Impossibile inserire il record di test",
	ErrDbRead:            "Impossibile leggere il record di test",
	ErrDbReadMismatch:    "Il valore del record di test non corrisponde a quello atteso",
	ErrDbUpdate:          "Impossibile aggiornare il record di test",
	ErrDbReadAfterUpdate: "Impossibile leggere il record di test dopo l'aggiornamento",
	ErrDbUpdateMismatch:  "Il valore del record aggiornato non corrisponde a quello atteso",

	// Server
	MsgServerStarting: "Avvio del server HTTP",
	MsgServerShutdown: "Arresto del server HTTP",
	MsgServerStopped:  "Server HTTP arrestato correttamente",
	MsgServerRoutes:   "Route registrate",
	ErrServerListen:   "Il server HTTP ha riscontrato un errore",
	ErrServerShutdown: "Il server HTTP è stato arrestato forzatamente",

	// Health
	MsgHealthReady:    "Tutti i servizi funzionanti",
	MsgHealthNotReady: "Uno o più servizi non pronti",

	// GitHub
	MsgGhConnecting:      "Connessione all'API GitHub",
	MsgGhConnected:       "Connesso all'API GitHub",
	MsgGhAuthToken:       "Autenticazione con token di accesso personale",
	MsgGhAuthApp:         "Autenticazione come GitHub App",
	MsgGhSelfTestPassed:  "Autotest GitHub superato",
	MsgGhRateRemaining:   "Stato del limite di frequenza dell'API GitHub",
	WarnGhRateLow:        "Il limite di frequenza dell'API GitHub sta per esaurirsi",
	ErrGhNotConfigured:   "GitHub non configurato — impostare un token o le credenziali dell'app",
	ErrGhAuth:            "Autenticazione GitHub fallita",
	ErrGhReadKey:         "Impossibile leggere il file della chiave privata della GitHub App",
	ErrGhParseKey:        "Impossibile analizzare la chiave privata della GitHub App",
	ErrGhInstallToken:    "Impossibile ottenere il token di installazione della GitHub App",
	ErrGhSelfTest:        "Autotest GitHub fallito",
	ErrGhRateLimit:       "Limite di frequenza dell'API GitHub troppo basso",
	ErrGhFetchUser:       "Impossibile recuperare l'utente GitHub",
	ErrGhFetchOrg:        "Impossibile recuperare l'organizzazione GitHub",
	ErrGhFetchRepo:       "Impossibile recuperare il repository GitHub",
	ErrGhFetchBranches:   "Impossibile recuperare i branch del repository",
	ErrGhFetchWorkflows:  "Impossibile recuperare i workflow del repository",
	ErrGhFetchRuns:       "Impossibile recuperare le esecuzioni dei workflow",
	ErrGhFetchRateLimit:  "Impossibile recuperare il limite di frequenza GitHub",
	ErrGhInvalidWorkflow: "ID workflow non valido",

	// GitHub collector
	MsgGhCollectorStarting:         "Avvio del collector dei repository GitHub",
	MsgGhCollectorTick:             "Esecuzione del ciclo di raccolta GitHub",
	MsgGhCollectorRepo:             "Raccolta dati per il repository",
	MsgGhCollectorStopped:          "Collector dei repository GitHub arrestato",
	MsgGhCollectorPassed:           "Autotest del collector GitHub superato",
	MsgGhCollectorAnnotations:      "Annotazioni raccolte per il job fallito",
	MsgGhCollectorLockAcquired:     "Lock di raccolta acquisito per il repository",
	MsgGhCollectorLockSkipped:      "Repository saltato (un'altra istanza detiene il lock)",
	WarnGhCollectorLockError:       "Impossibile acquisire il lock distribuito (si procede comunque)",
	ErrGhCollectorFetchRepo:        "Impossibile recuperare i metadati del repository",
	ErrGhCollectorFetchRuns:        "Impossibile recuperare le esecuzioni dei workflow",
	ErrGhCollectorFetchJobs:        "Impossibile recuperare i job dei workflow",
	ErrGhCollectorFetchAnnotations: "Impossibile recuperare le annotazioni del check-run",
	ErrGhCollectorNoRepos:          "Nessun repository configurato per il monitoraggio",
	ErrGhCollectorParseRepo:        "Formato del repository non valido (atteso: owner/repo)",

	// GitHub OAuth auth
	MsgAuthDevicePrompt:   "Inserisci il codice all'URL per autenticarti",
	MsgAuthPolling:        "In attesa dell'autorizzazione GitHub…",
	MsgAuthSuccess:        "Autenticazione con GitHub riuscita",
	MsgAuthLoggedOut:      "Credenziali GitHub rimosse",
	MsgAuthStatusLoggedIn: "Connesso a GitHub",
	MsgAuthStatusNoToken:  "Non connesso — esegui 'scuffinger github auth' per autenticarti",
	MsgAuthTokenFromVault: "Utilizzo del token GitHub dal vault di sistema",
	ErrAuthNoClientID:     "github.client_id deve essere impostato nella configurazione per il login OAuth",
	ErrAuthDeviceCode:     "Impossibile avviare il flusso dispositivo GitHub",
	ErrAuthPoll:           "Impossibile completare l'autorizzazione GitHub",
	ErrAuthSaveToken:      "Impossibile salvare il token nel vault di sistema",
	ErrAuthVerifyToken:    "Il token memorizzato non è più valido",

	// CLI commands
	CmdRootShort:         "Scuffinger — un servizio leggero di monitoraggio GitHub",
	CmdRootLong:          "Scuffinger è un servizio leggero per il monitoraggio di repository, workflow e metriche GitHub. Costruito con Cobra, Viper e Gin.",
	CmdVersionShort:      "Mostra il numero di versione",
	CmdVersionLong:       "Visualizza la versione corrente dell'applicazione scuffinger, incluse le informazioni di build quando disponibili.",
	CmdServeShort:        "Avvia il server HTTP",
	CmdServeLong:         "Avvia il server HTTP Gin con controlli di integrità, metriche Prometheus e endpoint API GitHub. Il server si connette a tutti i servizi di supporto configurati all'avvio.",
	CmdGitHubShort:       "Autenticazione e stato GitHub",
	CmdGitHubLong:        "Gestisci l'autenticazione OAuth GitHub. Usa i sottocomandi per accedere tramite flusso dispositivo, verificare lo stato dell'autenticazione o rimuovere le credenziali memorizzate.",
	CmdGitHubAuthShort:   "Autenticati con GitHub tramite OAuth",
	CmdGitHubAuthLong:    "Avvia il flusso dispositivo OAuth GitHub. Riceverai un codice monouso da inserire su github.com/login/device. Una volta autorizzato, il token viene memorizzato in modo sicuro nel vault di sistema.",
	CmdGitHubStatusShort: "Mostra lo stato attuale dell'autenticazione",
	CmdGitHubStatusLong:  "Visualizza lo stato attuale dell'autenticazione GitHub, incluso se le credenziali sono memorizzate nel file di configurazione, nelle variabili d'ambiente o nel vault di sistema.",
	CmdGitHubLogoutShort: "Rimuovi le credenziali GitHub memorizzate",
	CmdGitHubLogoutLong:  "Rimuove tutti i token OAuth GitHub memorizzati nel vault di sistema. Non influisce sui token configurati tramite file di configurazione o variabili d'ambiente.",
	CmdGitHubMonitorShort: "Avvia il collector dei repository GitHub",
	CmdGitHubMonitorLong:  "Avvia il collector GitHub in background che recupera periodicamente metadati dei repository, esecuzioni dei workflow e tempi dei passaggi dei job. Espone endpoint di integrità e metriche Prometheus, ma nessuna rotta proxy API. Progettato per essere eseguito come processo separato accanto al server API.",
	CmdFlagConfig:        "Percorso del file di configurazione",
}

