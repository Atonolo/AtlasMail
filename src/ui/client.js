'use strict';

(() => {
  const messages = {
    de: {
      skip: 'Zum Inhalt springen', workspace: 'ARBEITSBEREICH', workspaceName: 'Arbeitsbereich',
      overview: 'Übersicht', inbox: 'Posteingang', calendar: 'Kalender', tasks: 'Aufgaben', contacts: 'Kontakte', 
      themeToLight: 'Helles Design', themeToDark: 'Dunkles Design',
      themeActionLight: 'Helles Design aktivieren', themeActionDark: 'Dunkles Design aktivieren',
      privateTitle: 'Privat von Anfang an', privateDetail: 'Keine Telemetrie. Kein Cloud-Zwang.',
      foundation: 'PERSÖNLICHE E-MAIL', headline: 'Ihre E-Mail. Ihre Konten. Ihr Zuhause.',
      intro: 'Ein privater Ort für Ihre Kommunikation – offen und lokal gedacht.',
      noticeTitle: 'Direkte Verbindung zu Ihren Mailkonten',
      noticeText: 'Konten verbinden sich über den geschützten Dienst mit IMAP/SMTP oder JMAP. Nachrichten werden nur auf ausdrückliche Aktion geladen; externe Inhalte werden nicht nachgeladen.',
      webAccess: 'WEBZUGRIFF', loginTitle: 'Mit dem eigenen Dienst verbinden',
      loginText: 'Geben Sie das beim Start konfigurierte Zugriffstoken ein. Es wird nicht im Browser gespeichert.',
      tokenLabel: 'Zugriffstoken', connect: 'Verbinden', configuration: 'KONFIGURATION', accountsTitle: 'Persönliche Konten',
      accountsText: 'Verbindungsdaten liegen lokal; Zugangsdaten werden ausschließlich im OS-Schlüsselspeicher verwahrt.',
      emptyTitle: 'Noch keine Konten', emptyText: 'Fügen Sie ein persönliches IMAP/SMTP- oder JMAP-Konto hinzu.',
      addTitle: 'E-Mail-Konto verbinden', accountName: 'Anzeigename', accountAddress: 'E-Mail-Adresse', protocol: 'Protokoll',
      namePlaceholder: 'Privat', addressPlaceholder: 'name@example.org',
      username: 'Benutzername', authentication: 'Anmeldung', passwordAuth: 'Passwort / App-Passwort',
      password: 'Passwort oder App-Passwort', oauthDescription: 'Verwenden Sie OAuth-Endpunkte Ihres Anbieters. Die Anmeldung erfolgt mit Authorization Code und PKCE.',
      imapServer: 'IMAP-Server', smtpServer: 'SMTP-Server', server: 'Server', port: 'Port',
      tls: 'Verschlüsselung', jmapUrl: 'JMAP-Sitzungs-URL', authorizationUrl: 'Autorisierungs-URL',
      tokenUrl: 'Token-URL', clientId: 'Öffentliche Client-ID', oauthScope: 'Scopes',
      credentialNote: 'Zugangsdaten werden nicht exportiert oder protokolliert.', saveAccount: 'Konto sicher verbinden',
      accountConfigured: 'Konfiguriert', accountNeedsOAuth: 'Anmeldung ausstehend', removeAccount: 'Konto entfernen',
      mailbox: 'POSTEINGANG', mailTitle: 'Nachrichten', refreshMail: 'Nachrichten laden',
      mailEmpty: 'Richten Sie ein Konto ein und laden Sie Nachrichten.', composeTitle: 'Nachricht verfassen',
      sendFrom: 'Senden über dieses Konto', recipient: 'Empfänger', subject: 'Betreff',
      messageBody: 'Nachricht', htmlOptional: 'HTML-Version (optional)', attachments: 'Anhänge',
      send: 'Senden', reply: 'Antworten', downloadAttachment: 'Anhang herunterladen',
      availability: 'VERFÜGBARKEIT',       roadmapTitle: 'Weitere Funktionen',
      roadmapText: 'Kalender, Kontakte und Offline-Abgleich folgen in späteren Ausbaustufen.',
      roadmapCalendar: 'Kalender & Kontakte', roadmapCalendarDetail: 'CalDAV und CardDAV',
      roadmapSync: 'Offline & Abgleich', roadmapSyncDetail: 'Cache und Warteschlange', unavailable: 'Nicht verfügbar',
      foundationNote: 'Ihre Daten bleiben bei Ihnen. Der gemeinsame Kern benötigt kein Herstellerkonto und keinen Cloud-Dienst.',
      disconnect: 'Verbindung trennen', retry: 'Erneut versuchen', desktop: 'Lokale Anwendung', web: 'Eigener Webdienst',
      ready: 'Konten geladen.', saved: 'Konto sicher gespeichert.', disconnected: 'Verbindung getrennt.',
      oauthDone: 'OAuth-Anmeldung abgeschlossen.', messagesLoaded: 'Nachrichten geladen.',
      messageSent: 'Nachricht wurde gesendet.', accountRemoved: 'Konto und lokale Zugangsdaten wurden entfernt.',
      confirmRemove: 'Dieses Konto und seine Zugangsdaten wirklich entfernen?',
      saveRefreshFailed: 'Das Konto wurde gespeichert, aber die Anzeige konnte nicht aktualisiert werden.',
      error: 'Der Vorgang ist fehlgeschlagen.', AUTH_REQUIRED: 'Für dieses Konto ist eine Anmeldung erforderlich.',
      LOCAL_AUTH_REQUIRED: 'Die lokale Sitzung ist abgelaufen. Starten Sie die Anwendung neu und verwenden Sie deren frisch geöffnete Browserseite.',
      AUTH_EXPIRED: 'Die Anmeldung ist abgelaufen. Verbinden Sie das Konto erneut.',
      AUTH_FAILED: 'Anmeldung fehlgeschlagen. Prüfen Sie Benutzername, Passwort oder OAuth-Zugriff.',
      CONNECTION_FAILED: 'Der Mailserver ist nicht erreichbar.',
      STORE_LOCKED: 'Der Speicher wird gerade von einem anderen Prozess verwendet.',
      UNSUPPORTED_SCHEMA: 'Die Datenversion wird nicht unterstützt; vorhandene Daten wurden nicht verändert.',
      INVALID_STORED_DATA: 'Die gespeicherten Daten sind ungültig; vorhandene Daten wurden nicht verändert.',
      INVALID_ACCOUNT: 'Bitte geben Sie einen Anzeigenamen und eine gültige E-Mail-Adresse ein.',
      INVALID_MAIL_SERVER: 'Bitte prüfen Sie Servername, Port und TLS-Einstellung.',
      INVALID_JMAP_URL: 'Die JMAP-URL muss HTTPS verwenden (HTTP ist nur für localhost zulässig).',
      INVALID_OAUTH_CONFIGURATION: 'Bitte prüfen Sie OAuth-URLs, Client-ID und Scopes.',
      TLS_CERTIFICATE_INVALID: 'Das TLS-Zertifikat ist ungültig oder abgelaufen. Die Verbindung wurde abgebrochen.',
      TLS_REQUIRED: 'Der Server bietet keine sichere TLS-Verbindung an.',
      MAIL_RECEIVE_FAILED: 'Nachrichten konnten nicht sicher abgerufen werden.',
      SMTP_SEND_FAILED: 'Der SMTP-Server hat die Nachricht nicht angenommen.',
      SEND_STATUS_UNCERTAIN: 'Der Versandstatus ist unklar. Prüfen Sie den Gesendet-Ordner, bevor Sie erneut senden.',
      CREDENTIAL_STORE_UNAVAILABLE: 'Der OS-Schlüsselspeicher ist nicht verfügbar; das Konto wurde nicht eingerichtet.',
      CREDENTIAL_REVOKE_FAILED: 'Zugangsdaten konnten nicht sicher widerrufen werden. Das Konto bleibt erhalten.',
      OAUTH_TOKEN_EXCHANGE_FAILED: 'Der Anbieter konnte die OAuth-Anmeldung nicht abschließen.',
      OAUTH_STATE_INVALID: 'Die OAuth-Sitzung ist ungültig oder abgelaufen. Starten Sie die Anmeldung erneut.',
      ACCOUNT_NOT_CONFIGURED: 'Für dieses Konto fehlen Verbindungsdaten.',
      ACCOUNT_NOT_FOUND: 'Das Konto wurde nicht gefunden.',
      INVALID_MESSAGE: 'Bitte geben Sie einen Empfänger und einen Nachrichtentext ein.',
      INVALID_RECIPIENT: 'Bitte prüfen Sie die Empfängeradresse.',
      MESSAGE_TOO_LARGE: 'Die Nachricht oder ihre Anhänge sind zu groß.',
      OAUTH_POPUP_BLOCKED: 'Der Browser hat das OAuth-Fenster blockiert. Erlauben Sie Pop-ups und versuchen Sie es erneut.',
      ORIGIN_REJECTED: 'Der Zugriff wurde wegen einer ungültigen Adresse abgelehnt.',
      COMMIT_UNCERTAIN: 'Der Vorgang wurde möglicherweise gespeichert; prüfen Sie den Datenstand, bevor Sie ihn wiederholen.',
      READ_FAILED: 'Der Anwendungsspeicher konnte nicht gelesen werden.',
      WRITE_FAILED: 'Der Anwendungsspeicher konnte nicht geschrieben werden; der letzte Stand bleibt erhalten.',
      STORAGE_UNAVAILABLE: 'Der Anwendungsspeicher ist nicht verfügbar.',
    },
    en: {
      skip: 'Skip to content', workspace: 'WORKSPACE', workspaceName: 'Workspace',
      overview: 'Overview', inbox: 'Inbox', calendar: 'Calendar', tasks: 'Tasks', contacts: 'Contacts', 
      themeToLight: 'Light mode', themeToDark: 'Dark mode',
      themeActionLight: 'Enable light mode', themeActionDark: 'Enable dark mode',
      privateTitle: 'Private from the start', privateDetail: 'No telemetry. No cloud required.',
      foundation: 'PERSONAL EMAIL', headline: 'Your email. Your accounts. Your home.',
      intro: 'A private place for your communication, built to be open and local first.',
      noticeTitle: 'Connect directly to your mail accounts',
      noticeText: 'Accounts connect through the protected service using IMAP/SMTP or JMAP. Messages load only when requested; remote content is never loaded automatically.',
      webAccess: 'WEB ACCESS', loginTitle: 'Connect to your own service',
      loginText: 'Enter the access token configured at startup. It is not stored in your browser.',
      tokenLabel: 'Access token', connect: 'Connect', configuration: 'CONFIGURATION', accountsTitle: 'Personal accounts',
      accountsText: 'Connection settings stay local; credentials are kept only in the operating system credential store.',
      emptyTitle: 'No accounts yet', emptyText: 'Add a personal IMAP/SMTP or JMAP account.',
      addTitle: 'Connect an email account', accountName: 'Display name', accountAddress: 'Email address', protocol: 'Protocol',
      namePlaceholder: 'Personal', addressPlaceholder: 'name@example.org',
      username: 'Username', authentication: 'Authentication', passwordAuth: 'Password / app password',
      password: 'Password or app password', oauthDescription: 'Enter your provider OAuth endpoints. Sign-in uses Authorization Code with PKCE.',
      imapServer: 'IMAP server', smtpServer: 'SMTP server', server: 'Server', port: 'Port',
      tls: 'Encryption', jmapUrl: 'JMAP session URL', authorizationUrl: 'Authorization URL',
      tokenUrl: 'Token URL', clientId: 'Public client ID', oauthScope: 'Scopes',
      credentialNote: 'Credentials are never exported or logged.', saveAccount: 'Connect account securely',
      accountConfigured: 'Configured', accountNeedsOAuth: 'Sign-in required', removeAccount: 'Remove account',
      mailbox: 'INBOX', mailTitle: 'Messages', refreshMail: 'Load messages',
      mailEmpty: 'Add an account and load its messages.', composeTitle: 'Compose message',
      sendFrom: 'Send from this account', recipient: 'Recipient', subject: 'Subject',
      messageBody: 'Message', htmlOptional: 'HTML version (optional)', attachments: 'Attachments',
      send: 'Send', reply: 'Reply', downloadAttachment: 'Download attachment',
      availability: 'AVAILABILITY',       roadmapTitle: 'More features',
      roadmapText: 'Calendar, contacts, and offline sync are planned for later stages.',
      roadmapCalendar: 'Calendar & contacts', roadmapCalendarDetail: 'CalDAV and CardDAV',
      roadmapSync: 'Offline & sync', roadmapSyncDetail: 'Cache and queue', unavailable: 'Unavailable',
      foundationNote: 'Your data stays with you. The shared core does not require a vendor account or cloud service.',
      disconnect: 'Disconnect', retry: 'Try again', desktop: 'Local application', web: 'Your web service',
      ready: 'Accounts loaded.', saved: 'Account saved securely.', disconnected: 'Disconnected.',
      oauthDone: 'OAuth sign-in completed.', messagesLoaded: 'Messages loaded.',
      messageSent: 'Message sent.', accountRemoved: 'Account and local credentials removed.',
      confirmRemove: 'Remove this account and its credentials?',
      saveRefreshFailed: 'The account was saved, but the display could not be refreshed.',
      error: 'The operation failed.', AUTH_REQUIRED: 'This account needs sign-in.',
      LOCAL_AUTH_REQUIRED: 'The local session has expired. Restart the application and use its newly opened browser page.',
      AUTH_EXPIRED: 'Sign-in expired. Reconnect this account.',
      AUTH_FAILED: 'Sign-in failed. Check the username, password, or OAuth access.',
      CONNECTION_FAILED: 'The mail server could not be reached.',
      STORE_LOCKED: 'The store is in use by another process.',
      UNSUPPORTED_SCHEMA: 'The data version is unsupported; existing data was not changed.',
      INVALID_STORED_DATA: 'The stored data is invalid; existing data was not changed.',
      INVALID_ACCOUNT: 'Enter a display name and valid email address.',
      INVALID_MAIL_SERVER: 'Check the server name, port, and TLS setting.',
      INVALID_JMAP_URL: 'The JMAP URL must use HTTPS (HTTP is allowed only for localhost).',
      INVALID_OAUTH_CONFIGURATION: 'Check the OAuth URLs, client ID, and scopes.',
      TLS_CERTIFICATE_INVALID: 'The TLS certificate is invalid or expired. The connection was closed.',
      TLS_REQUIRED: 'The server did not offer a secure TLS connection.',
      MAIL_RECEIVE_FAILED: 'Messages could not be retrieved securely.',
      SMTP_SEND_FAILED: 'The SMTP server did not accept the message.',
      SEND_STATUS_UNCERTAIN: 'Delivery status is uncertain. Check Sent before trying again.',
      CREDENTIAL_STORE_UNAVAILABLE: 'The OS credential store is unavailable; the account was not configured.',
      CREDENTIAL_REVOKE_FAILED: 'Credentials could not be safely revoked. The account was kept.',
      OAUTH_TOKEN_EXCHANGE_FAILED: 'The provider could not complete OAuth sign-in.',
      OAUTH_STATE_INVALID: 'The OAuth session is invalid or expired. Start sign-in again.',
      ACCOUNT_NOT_CONFIGURED: 'Connection settings are missing for this account.',
      ACCOUNT_NOT_FOUND: 'The account was not found.',
      INVALID_MESSAGE: 'Enter a recipient and message text.',
      INVALID_RECIPIENT: 'Check the recipient address.',
      MESSAGE_TOO_LARGE: 'The message or its attachments are too large.',
      OAUTH_POPUP_BLOCKED: 'The browser blocked the OAuth window. Allow pop-ups and try again.',
      ORIGIN_REJECTED: 'The request was rejected because its origin is invalid.',
      COMMIT_UNCERTAIN: 'The operation may have been saved; check stored data before repeating it.',
      READ_FAILED: 'The application store could not be read.',
      WRITE_FAILED: 'The application store could not be written; the previous state remains intact.',
      STORAGE_UNAVAILABLE: 'The application store is unavailable.',
    },
  };
  const $ = (id) => document.getElementById(id);
  const desktop = window.location.pathname === '/desktop';
  let language = navigator.language.toLowerCase().startsWith('de') ? 'de' : 'en';
  let theme = 'dark';
  let token = '';
  let connected = desktop;
  let connectionVersion = 0;
  let state = null;
  let currentMessages = [];
  let currentAccountId = '';
  let statusKey = '';
  let errorCode = '';
  let busy = false;
  let oauthPopup = null;

  function translate(key) {
    return messages[language][key] || key;
  }

  function loadTheme() {
    try {
      return window.localStorage.getItem('atlas-mail-theme') === 'light' ? 'light' : 'dark';
    } catch (error) {
      if (error?.name === 'SecurityError') return 'dark';
      throw error;
    }
  }

  function renderTheme() {
    document.documentElement.dataset.theme = theme;
    const label = translate(theme === 'dark' ? 'themeToLight' : 'themeToDark');
    const action = translate(theme === 'dark' ? 'themeActionLight' : 'themeActionDark');
    $('theme-label').textContent = label;
    $('theme-icon').textContent = theme === 'dark' ? '☼' : '☾';
    $('theme-toggle').setAttribute('aria-label', action);
    $('theme-toggle').title = action;
  }

  function renderLanguage() {
    document.documentElement.lang = language;
    document.querySelectorAll('[data-i18n]').forEach((element) => {
      element.textContent = translate(element.dataset.i18n);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach((element) => {
      element.placeholder = translate(element.dataset.i18nPlaceholder);
    });
    $('language-de').setAttribute('aria-pressed', String(language === 'de'));
    $('language-en').setAttribute('aria-pressed', String(language === 'en'));
    $('environment-label').textContent = translate(desktop ? 'desktop' : 'web');
    $('status').textContent = statusKey ? translate(statusKey) : '';
    $('error').textContent = errorCode
      ? `${translate(errorCode in messages[language] ? errorCode : 'error')} (${errorCode})`
      : '';
  }

  function toggleAccountFields() {
    const jmap = $('account-protocol').value === 'jmap';
    const oauth = $('auth-mechanism').value === 'oauth2';
    $('imap-fields').hidden = jmap;
    $('jmap-fields').hidden = !jmap;
    $('jmap-url').required = jmap;
    $('oauth-fields').hidden = !oauth;
    $('secret-field').hidden = oauth;
    for (const control of $('imap-fields').querySelectorAll('input,select')) control.disabled = jmap || !connected || busy;
    $('jmap-url').disabled = !jmap || !connected || busy;
    for (const control of $('oauth-fields').querySelectorAll('input')) control.disabled = !oauth || !connected || busy;
    $('account-secret').required = !oauth;
    for (const control of $('oauth-fields').querySelectorAll('input[type=url],input[type=text]')) {
      control.required = oauth;
    }
  }

  function renderState() {
    const accounts = state ? state.accounts : [];
    const readyAccounts = accounts.filter((account) => account.configured);
    $('account-count').textContent = String(accounts.length);
    $('account-empty').hidden = accounts.length > 0;
    const list = $('account-list');
    list.replaceChildren();
    for (const account of accounts) {
      const item = document.createElement('li');
      const avatar = document.createElement('span');
      avatar.className = 'account-avatar';
      avatar.setAttribute('aria-hidden', 'true');
      avatar.textContent = account.label.slice(0, 1).toUpperCase();
      const information = document.createElement('span');
      information.className = 'account-info';
      const name = document.createElement('strong');
      name.textContent = account.label;
      const address = document.createElement('small');
      address.textContent = account.address;
      const accountState = document.createElement('small');
      accountState.className = account.configured ? 'account-configured' : 'account-unconfigured';
      accountState.textContent = translate(account.configured ? 'accountConfigured' : 'accountNeedsOAuth');
      information.append(name, address, accountState);
      const protocol = document.createElement('span');
      protocol.className = 'protocol-tag';
      protocol.textContent = account.protocol === 'jmap' ? 'JMAP' : 'IMAP / SMTP';
      const remove = document.createElement('button');
      remove.type = 'button';
      remove.className = 'button button-secondary account-remove';
      remove.textContent = translate('removeAccount');
      remove.setAttribute('aria-label', `${translate('removeAccount')}: ${account.label}`);
      remove.disabled = busy;
      remove.addEventListener('click', () => removeAccount(account));
      item.append(avatar, information, protocol, remove);
      list.append(item);
    }

    const select = $('compose-account');
    const selected = select.value || currentAccountId;
    select.replaceChildren();
    for (const account of readyAccounts) {
      const option = document.createElement('option');
      option.value = account.id;
      option.textContent = `${account.label} <${account.address}>`;
      select.append(option);
    }
    if (readyAccounts.some((account) => account.id === selected)) select.value = selected;
    else if (readyAccounts.length) select.value = readyAccounts[0].id;
    currentAccountId = select.value;

    $('login-section').hidden = desktop || connected;
    $('disconnect').hidden = desktop || !connected;
    $('dashboard').hidden = !connected || !state;
    $('retry').hidden = !(connected && errorCode);
    $('mail-panel').hidden = !connected || !state;
    $('refresh-mail').disabled = !connected || !state || readyAccounts.length === 0 || busy;
    $('compose-panel').hidden = !connected || !state || readyAccounts.length === 0;
    $('account-form').querySelectorAll('input,select,button').forEach((control) => {
      control.disabled = !connected || !state || busy;
    });
    $('compose-form').querySelectorAll('input,select,textarea,button').forEach((control) => {
      control.disabled = !connected || !state || readyAccounts.length === 0 || busy;
    });
    toggleAccountFields();
    const selectedAccount = state?.accounts.find((account) => account.id === currentAccountId);
    $('sender-confirmation').textContent = selectedAccount
      ? `${translate('sendFrom')}: ${selectedAccount.label} <${selectedAccount.address}>`
      : '';
    renderMessages();
  }

  function setStatus(key = '', code = '') {
    statusKey = key;
    errorCode = code;
    $('error').hidden = !code;
    renderLanguage();
    renderState();
  }

  async function request(operation, input) {
    try {
      const options = { headers: { Authorization: `Bearer ${token}` }, cache: 'no-store' };
      if (desktop) options.headers = { 'X-Atlas-Local': '1' };
      let endpoint = '/api/v1/status';
      if (operation === 'setup') {
        endpoint = '/api/v1/accounts/setup';
        options.method = 'POST';
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(input);
      } else if (operation === 'send') {
        endpoint = '/api/v1/mail/send';
        options.method = 'POST';
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(input);
      } else if (operation.startsWith('mail:')) {
        endpoint = `/api/v1/mail?accountId=${encodeURIComponent(operation.slice(5))}`;
      } else if (operation.startsWith('remove:')) {
        endpoint = `/api/v1/accounts/${encodeURIComponent(operation.slice(7))}`;
        options.method = 'DELETE';
      } else if (operation.startsWith('oauth:')) {
        endpoint = `/api/v1/oauth/start?accountId=${encodeURIComponent(operation.slice(6))}`;
      }
      const response = await fetch(endpoint, options);
      const result = await response.json();
      if (!response.ok && result.ok !== false) return { ok: false, code: 'UNEXPECTED_ERROR' };
      return result;
    } catch {
      return { ok: false, code: 'CONNECTION_FAILED' };
    }
  }

  async function connectLocalSession() {
    let launchToken = new URLSearchParams(window.location.hash.slice(1)).get('session') || '';
    if (launchToken) {
      history.replaceState(null, '', window.location.pathname);
      try {
        const response = await fetch('/api/v1/session', {
          method: 'POST',
          headers: { Authorization: `Bearer ${launchToken}` },
          cache: 'no-store',
        });
        launchToken = '';
        if (!response.ok) {
          setStatus('', 'LOCAL_AUTH_REQUIRED');
          return;
        }
      } catch {
        launchToken = '';
        setStatus('', 'CONNECTION_FAILED');
        return;
      }
    }
    await refresh();
  }

  async function refresh(successKey = 'ready', version = connectionVersion) {
    const result = await request('status');
    if (version !== connectionVersion) return false;
    if (!result.ok) {
      setStatus('', result.code || 'UNEXPECTED_ERROR');
      return false;
    }
    state = result.data;
    connected = true;
    setStatus(successKey);
    return true;
  }

  async function removeAccount(account) {
    if (busy || !window.confirm(translate('confirmRemove'))) return;
    busy = true;
    renderState();
    const result = await request(`remove:${account.id}`);
    if (!result.ok) {
      setStatus('', result.code || 'UNEXPECTED_ERROR');
    } else {
      if (currentAccountId === account.id) {
        currentAccountId = '';
        currentMessages = [];
      }
      await refresh('accountRemoved');
    }
    busy = false;
    renderState();
  }

  function htmlToText(html) {
    return String(html || '')
      .replace(/<(script|style|iframe|object|svg)[^>]*>[\s\S]*?<\/\1\s*>/gi, ' ')
      .replace(/<[^>]*>/g, ' ')
      .replace(/&nbsp;/gi, ' ')
      .replace(/&amp;/gi, '&')
      .replace(/&lt;/gi, '<')
      .replace(/&gt;/gi, '>')
      .replace(/&quot;/gi, '"')
      .replace(/&#39;/gi, "'")
      .replace(/\s+/g, ' ')
      .trim();
  }

  function renderMessages() {
    const list = $('message-list');
    list.replaceChildren();
    $('mail-empty').hidden = currentMessages.length > 0;
    for (const message of currentMessages) {
      const item = document.createElement('li');
      const article = document.createElement('article');
      article.className = 'mail-message';
      const heading = document.createElement('h3');
      heading.textContent = message.subject || '(no subject)';
      const from = document.createElement('p');
      from.className = 'mail-message-from';
      from.textContent = message.from || '';
      const date = document.createElement('time');
      date.textContent = message.date ? new Date(message.date).toLocaleString() : '';
      const body = document.createElement('pre');
      body.className = 'mail-message-body';
      body.textContent = message.text || htmlToText(message.html);
      article.append(heading, from, date, body);
      if (message.attachments?.length) {
        const attachments = document.createElement('div');
        attachments.className = 'mail-attachments';
        for (const attachment of message.attachments) {
          const binary = atob(attachment.data || '');
          const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
          const blob = new Blob([bytes], { type: attachment.contentType || 'application/octet-stream' });
          const link = document.createElement('a');
          link.href = URL.createObjectURL(blob);
          link.download = attachment.name || 'attachment';
          link.textContent = `${translate('downloadAttachment')}: ${attachment.name || 'attachment'}`;
          attachments.append(link);
        }
        article.append(attachments);
      }
      if (message.messageId) {
        const reply = document.createElement('button');
        reply.type = 'button';
        reply.className = 'button button-secondary';
        reply.textContent = translate('reply');
        reply.addEventListener('click', () => replyTo(message));
        article.append(reply);
      }
      item.append(article);
      list.append(item);
    }
  }

  function replyTo(message) {
    const address = message.from.match(/<([^>]+)>/)?.[1] || message.from;
    $('compose-account').value = message.accountId;
    currentAccountId = message.accountId;
    $('compose-to').value = address;
    $('compose-subject').value = /^re:/i.test(message.subject) ? message.subject : `Re: ${message.subject}`;
    $('compose-body').value = `\n\n> ${(message.text || htmlToText(message.html)).split('\n').join('\n> ')}`;
    $('reply-in-reply-to').value = message.messageId || '';
    $('reply-references').value = message.references || message.messageId || '';
    renderState();
    $('compose-panel').scrollIntoView({ behavior: 'smooth', block: 'start' });
    $('compose-to').focus();
  }

  async function readAttachments(files) {
    const attachments = [];
    let totalSize = 0;
    for (const file of files) {
      totalSize += file.size;
      if (file.size > 10 * 1024 * 1024 || totalSize > 11 * 1024 * 1024) {
        throw new Error('MESSAGE_TOO_LARGE');
      }
      const bytes = new Uint8Array(await file.arrayBuffer());
      let binary = '';
      for (let offset = 0; offset < bytes.length; offset += 0x8000) {
        binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
      }
      attachments.push({ name: file.name, contentType: file.type, data: btoa(binary) });
    }
    return attachments;
  }

  $('language-de').addEventListener('click', () => { language = 'de'; renderLanguage(); renderState(); });
  $('language-en').addEventListener('click', () => { language = 'en'; renderLanguage(); renderState(); });
  $('theme-toggle').addEventListener('click', () => {
    theme = theme === 'dark' ? 'light' : 'dark';
    renderTheme();
    try {
      window.localStorage.setItem('atlas-mail-theme', theme);
    } catch (error) {
      if (error?.name !== 'SecurityError' && error?.name !== 'QuotaExceededError') throw error;
    }
  });
  $('login-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    const version = ++connectionVersion;
    token = $('token').value;
    $('token').value = '';
    if (!(await refresh('ready', version)) && version === connectionVersion) token = '';
    renderState();
  });
  $('disconnect').addEventListener('click', () => {
    connectionVersion += 1;
    token = '';
    connected = false;
    state = null;
    currentMessages = [];
    setStatus('disconnected');
  });
  $('retry').addEventListener('click', () => refresh());
  $('account-protocol').addEventListener('change', toggleAccountFields);
  $('auth-mechanism').addEventListener('change', toggleAccountFields);
  $('compose-account').addEventListener('change', () => {
    currentAccountId = $('compose-account').value;
    renderState();
  });
  $('account-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    if (busy || !connected) return;
    const useOAuth = $('auth-mechanism').value === 'oauth2';
    oauthPopup = useOAuth ? window.open('about:blank', 'atlas-mail-oauth') : null;
    busy = true;
    renderState();
    const version = connectionVersion;
    const protocol = $('account-protocol').value;
    const profile = {
      username: $('account-username').value,
      authMechanism: $('auth-mechanism').value,
      imap: protocol === 'imap-smtp'
        ? { host: $('imap-host').value, port: Number($('imap-port').value), tlsMode: $('imap-tls').value }
        : { host: '', port: 0, tlsMode: '' },
      smtp: protocol === 'imap-smtp'
        ? { host: $('smtp-host').value, port: Number($('smtp-port').value), tlsMode: $('smtp-tls').value }
        : { host: '', port: 0, tlsMode: '' },
      jmapUrl: protocol === 'jmap' ? $('jmap-url').value : '',
      oauth: useOAuth ? {
        authorizationUrl: $('oauth-authorization-url').value,
        tokenUrl: $('oauth-token-url').value,
        clientId: $('oauth-client-id').value,
        scope: $('oauth-scope').value,
      } : null,
    };
    const result = await request('setup', {
      account: {
        label: $('account-name').value,
        address: $('account-address').value,
        protocol,
      },
      profile,
      secret: useOAuth ? '' : $('account-secret').value,
    });
    if (version !== connectionVersion) {
      if (oauthPopup) oauthPopup.close();
      busy = false;
      renderState();
      return;
    }
    if (!result.ok) {
      if (oauthPopup) oauthPopup.close();
      setStatus('', result.code || 'UNEXPECTED_ERROR');
    } else {
      $('account-form').reset();
      const newAccountID = result.data.id;
      if (!(await refresh('saved', version)) && version === connectionVersion) {
        setStatus('', 'saveRefreshFailed');
      }
      if (useOAuth && oauthPopup) {
        const oauth = await request(`oauth:${newAccountID}`);
        if (oauth.ok) oauthPopup.location = oauth.data.authorizationUrl;
        else {
          oauthPopup.close();
          setStatus('', oauth.code || 'OAUTH_TOKEN_EXCHANGE_FAILED');
        }
      } else if (useOAuth) {
        setStatus('', 'OAUTH_POPUP_BLOCKED');
      }
    }
    busy = false;
    renderState();
  });
  $('refresh-mail').addEventListener('click', async () => {
    const accountID = $('compose-account').value;
    if (!accountID || busy) return;
    busy = true;
    renderState();
    const result = await request(`mail:${accountID}`);
    if (!result.ok) setStatus('', result.code || 'MAIL_RECEIVE_FAILED');
    else {
      currentAccountId = accountID;
      currentMessages = result.data || [];
      setStatus('messagesLoaded');
    }
    busy = false;
    renderState();
  });
  $('compose-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    if (busy || !connected) return;
    const text = $('compose-body').value;
    const html = $('compose-html').value;
    if (!text && !html) {
      setStatus('', 'INVALID_MESSAGE');
      return;
    }
    let attachments;
    try {
      attachments = await readAttachments(Array.from($('compose-attachments').files || []));
    } catch {
      setStatus('', 'MESSAGE_TOO_LARGE');
      return;
    }
    busy = true;
    renderState();
    const result = await request('send', {
      accountId: $('compose-account').value,
      to: $('compose-to').value.split(',').map((recipient) => recipient.trim()).filter(Boolean),
      subject: $('compose-subject').value,
      text,
      html,
      attachments,
      inReplyTo: $('reply-in-reply-to').value,
      references: $('reply-references').value,
    });
    if (!result.ok) setStatus('', result.code || 'SMTP_SEND_FAILED');
    else {
      $('compose-form').reset();
      $('reply-in-reply-to').value = '';
      $('reply-references').value = '';
      setStatus('messageSent');
    }
    busy = false;
    renderState();
  });
  window.addEventListener('message', async (event) => {
    if (event.origin !== window.location.origin || event.source !== oauthPopup ||
        event.data?.type !== 'atlas-mail-oauth-complete') return;
    oauthPopup.close();
    oauthPopup = null;
    if (event.data.success) await refresh('oauthDone');
    else setStatus('', 'OAUTH_TOKEN_EXCHANGE_FAILED');
  });

  theme = loadTheme();
  renderTheme();
  renderLanguage();
  renderState();
  if (desktop) connectLocalSession();
})();
