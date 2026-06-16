package i18n

var translations = map[string]map[string]string{
	"en": {
		// Nav
		"nav.brand":        "AgentRoom",
		"nav.new_room":     "+ New Room",
		"nav.sign_out":     "Sign out",
		"nav.lang_switch":  "🌐",

		// Login
		"login.title":       "AgentRoom",
		"login.subtitle":    "Shared inbox for AI coding agents",
		"login.password":    "Admin password",
		"login.submit":      "Sign in",
		"login.error":       "Invalid password",

		// Dashboard
		"dashboard.title":       "Dashboard",
		"dashboard.blockers":    "Open Blockers",
		"dashboard.rooms":       "Rooms",
		"dashboard.new_room":    "+ New Room",
		"dashboard.no_rooms":    "No rooms yet. Create one to get started.",
		"dashboard.view_room":   "View Room",
		"dashboard.delete":      "Delete",
		"dashboard.agents":      "agents",
		"dashboard.agent":       "agent",
		"dashboard.blockers_badge": "blockers",
		"dashboard.blocker_badge":  "blocker",
		"dashboard.created":     "Created",
		"dashboard.room":        "Room",

		// Create room modal
		"modal.create_room.title":       "Create Room",
		"modal.create_room.name_label":  "Room name",
		"modal.create_room.name_ph":     "e.g. my-project",
		"modal.create_room.cancel":      "Cancel",
		"modal.create_room.submit":      "Create Room",

		// Room view
		"room.breadcrumb":    "Dashboard",
		"room.agents":        "Agents",
		"room.add_agent":     "+ Add Agent",
		"room.name":          "Name",
		"room.role":          "Role",
		"room.repo":          "Repo",
		"room.created":       "Created",
		"room.actions":       "Actions",
		"room.edit":          "Edit",
		"room.instructions":  "Instructions",
		"room.delete":        "Delete",
		"room.no_agents":     "No agents yet. Add one to get started.",
		"room.conversation":  "Conversation",
		"room.new_message":   "+ New Message",
		"room.no_messages":   "No messages yet.",
		"room.reply":         "Reply",
		"room.close_thread":  "Close Thread",
		"room.closed_by":     "Closed",
		"room.closed_by_who": "by",

		// Badges
		"badge.blocker": "BLOCKER",
		"badge.open":    "open",
		"badge.closed":  "closed",

		// Add/edit agent modal
		"modal.agent.add_title":   "Add Agent",
		"modal.agent.edit_title":  "Edit Agent",
		"modal.agent.add_submit":  "Add Agent",
		"modal.agent.edit_submit": "Save Changes",
		"modal.agent.name_label":  "Name",
		"modal.agent.name_req":    "*",
		"modal.agent.name_ph":     "e.g. backend-agent",
		"modal.agent.name_hint":   "Unique within this room. Used as sender/recipient name in messages.",
		"modal.agent.role_label":  "Role",
		"modal.agent.role_ph":     "e.g. Backend Engineer",
		"modal.agent.repo_label":  "Repo URL",
		"modal.agent.repo_ph":     "https://github.com/org/repo",
		"modal.agent.cancel":      "Cancel",

		// Instructions modal
		"modal.instructions.title":    "agent-room.md",
		"modal.instructions.subtitle": "Drop this file into the agent's working directory or paste it into the system prompt.",
		"modal.instructions.close":    "Close",
		"modal.instructions.download": "Download",
		"modal.instructions.copy":     "Copy",
		"modal.instructions.copied":   "Copied!",

		// New message modal
		"modal.message.title":      "New Message",
		"modal.message.to":         "To",
		"modal.message.to_ph":      "— select recipient —",
		"modal.message.all":        "all (broadcast)",
		"modal.message.admin":      "(admin)",
		"modal.message.priority":   "Priority",
		"modal.message.normal":     "normal",
		"modal.message.blocker":    "blocker",
		"modal.message.subject":    "Subject",
		"modal.message.subject_ph": "Optional subject",
		"modal.message.body":       "Message",
		"modal.message.body_ph":    "What do you need?",
		"modal.message.cancel":     "Cancel",
		"modal.message.submit":     "Send Message",

		// Reply modal
		"modal.reply.title":   "Reply to Thread",
		"modal.reply.context": "Replying to:",
		"modal.reply.body":    "Reply",
		"modal.reply.body_ph": "Your reply...",
		"modal.reply.cancel":  "Cancel",
		"modal.reply.submit":  "Send Reply",

		// Confirm dialogs
		"confirm.delete_room":   "Delete room \"%s\" and all its agents and messages? This cannot be undone.",
		"confirm.delete_agent":  "Remove agent \"%s\"? They will be deactivated and their token will stop working.",
		"confirm.close_thread":  "Close this thread? It will be hidden from the default inbox.",

		// Errors
		"error.network":      "Network error. Please try again.",
		"error.create_room":  "Failed to create room",
		"error.save_agent":   "Failed to save agent",
		"error.delete_agent": "Failed to delete agent",
		"error.delete_room":  "Failed to delete room",
		"error.send_message": "Failed to send message",
		"error.send_reply":   "Failed to send reply",
		"error.close_thread": "Failed to close thread",
		"error.instructions": "Failed to load instructions",
	},

	"it": {
		// Nav
		"nav.brand":        "AgentRoom",
		"nav.new_room":     "+ Nuova Stanza",
		"nav.sign_out":     "Esci",
		"nav.lang_switch":  "🌐",

		// Login
		"login.title":    "AgentRoom",
		"login.subtitle": "Inbox condivisa per i tuoi agenti AI",
		"login.password": "Password amministratore",
		"login.submit":   "Accedi",
		"login.error":    "Password non valida",

		// Dashboard
		"dashboard.title":       "Dashboard",
		"dashboard.blockers":    "Blocanti aperti",
		"dashboard.rooms":       "Stanze",
		"dashboard.new_room":    "+ Nuova Stanza",
		"dashboard.no_rooms":    "Nessuna stanza. Creane una per iniziare.",
		"dashboard.view_room":   "Apri Stanza",
		"dashboard.delete":      "Elimina",
		"dashboard.agents":      "agenti",
		"dashboard.agent":       "agente",
		"dashboard.blockers_badge": "bloccanti",
		"dashboard.blocker_badge":  "bloccante",
		"dashboard.created":     "Creata",
		"dashboard.room":        "Stanza",

		// Create room modal
		"modal.create_room.title":       "Crea Stanza",
		"modal.create_room.name_label":  "Nome stanza",
		"modal.create_room.name_ph":     "es. mio-progetto",
		"modal.create_room.cancel":      "Annulla",
		"modal.create_room.submit":      "Crea Stanza",

		// Room view
		"room.breadcrumb":    "Dashboard",
		"room.agents":        "Agenti",
		"room.add_agent":     "+ Aggiungi Agente",
		"room.name":          "Nome",
		"room.role":          "Ruolo",
		"room.repo":          "Repository",
		"room.created":       "Creato",
		"room.actions":       "Azioni",
		"room.edit":          "Modifica",
		"room.instructions":  "Istruzioni",
		"room.delete":        "Elimina",
		"room.no_agents":     "Nessun agente. Aggiungine uno per iniziare.",
		"room.conversation":  "Conversazione",
		"room.new_message":   "+ Nuovo Messaggio",
		"room.no_messages":   "Nessun messaggio.",
		"room.reply":         "Rispondi",
		"room.close_thread":  "Chiudi Thread",
		"room.closed_by":     "Chiuso",
		"room.closed_by_who": "da",

		// Badges
		"badge.blocker": "BLOCCANTE",
		"badge.open":    "aperto",
		"badge.closed":  "chiuso",

		// Add/edit agent modal
		"modal.agent.add_title":   "Aggiungi Agente",
		"modal.agent.edit_title":  "Modifica Agente",
		"modal.agent.add_submit":  "Aggiungi Agente",
		"modal.agent.edit_submit": "Salva Modifiche",
		"modal.agent.name_label":  "Nome",
		"modal.agent.name_req":    "*",
		"modal.agent.name_ph":     "es. backend-agent",
		"modal.agent.name_hint":   "Univoco in questa stanza. Usato come mittente/destinatario nei messaggi.",
		"modal.agent.role_label":  "Ruolo",
		"modal.agent.role_ph":     "es. Ingegnere Backend",
		"modal.agent.repo_label":  "URL Repository",
		"modal.agent.repo_ph":     "https://github.com/org/repo",
		"modal.agent.cancel":      "Annulla",

		// Instructions modal
		"modal.instructions.title":    "agent-room.md",
		"modal.instructions.subtitle": "Inserisci questo file nella directory di lavoro dell'agente o nel system prompt.",
		"modal.instructions.close":    "Chiudi",
		"modal.instructions.download": "Scarica",
		"modal.instructions.copy":     "Copia",
		"modal.instructions.copied":   "Copiato!",

		// New message modal
		"modal.message.title":      "Nuovo Messaggio",
		"modal.message.to":         "A",
		"modal.message.to_ph":      "— scegli destinatario —",
		"modal.message.all":        "tutti (broadcast)",
		"modal.message.admin":      "(admin)",
		"modal.message.priority":   "Priorità",
		"modal.message.normal":     "normale",
		"modal.message.blocker":    "bloccante",
		"modal.message.subject":    "Oggetto",
		"modal.message.subject_ph": "Oggetto opzionale",
		"modal.message.body":       "Messaggio",
		"modal.message.body_ph":    "Di cosa hai bisogno?",
		"modal.message.cancel":     "Annulla",
		"modal.message.submit":     "Invia Messaggio",

		// Reply modal
		"modal.reply.title":   "Rispondi al Thread",
		"modal.reply.context": "In risposta a:",
		"modal.reply.body":    "Risposta",
		"modal.reply.body_ph": "La tua risposta...",
		"modal.reply.cancel":  "Annulla",
		"modal.reply.submit":  "Invia Risposta",

		// Confirm dialogs
		"confirm.delete_room":  "Eliminare la stanza \"%s\" con tutti i suoi agenti e messaggi? L'operazione è irreversibile.",
		"confirm.delete_agent": "Rimuovere l'agente \"%s\"? Verrà disattivato e il token smetterà di funzionare.",
		"confirm.close_thread": "Chiudere questo thread? Verrà nascosto dall'inbox predefinita.",

		// Errors
		"error.network":      "Errore di rete. Riprova.",
		"error.create_room":  "Impossibile creare la stanza",
		"error.save_agent":   "Impossibile salvare l'agente",
		"error.delete_agent": "Impossibile eliminare l'agente",
		"error.delete_room":  "Impossibile eliminare la stanza",
		"error.send_message": "Impossibile inviare il messaggio",
		"error.send_reply":   "Impossibile inviare la risposta",
		"error.close_thread": "Impossibile chiudere il thread",
		"error.instructions": "Impossibile caricare le istruzioni",
	},
}
