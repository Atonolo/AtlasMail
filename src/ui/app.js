'use strict';

(() => {
  const messages = {
    de: {
      skip: 'Zum Inhalt springen', workspace: 'ARBEITSBEREICH', workspaceName: 'Arbeitsbereich',
      overview: 'Übersicht', inbox: 'Posteingang', calendar: 'Kalender', tasks: 'Aufgaben', contacts: 'Kontakte', later: 'SPÄTER',
      privateTitle: 'Privat von Anfang an', privateDetail: 'Keine Telemetrie. Kein Cloud-Zwang.',
      foundation: 'DAS FUNDAMENT', headline: 'Willkommen bei Atlas Mail.',
      intro: 'Ein privater Ort für Ihre Kommunikation – von Grund auf offen und lokal gedacht.',
      noticeTitle: 'Das Grundgerüst ist noch kein Mailprogramm',
      noticeText: 'Hier können Sie ausschließlich synthetische Testkonten speichern. E-Mail, Synchronisierung und Kalender sind noch nicht verfügbar. Bitte keine echten Zugangsdaten eingeben.',
      webAccess: 'WEBZUGRIFF', loginTitle: 'Mit dem eigenen Dienst verbinden',
      loginText: 'Geben Sie das beim Start konfigurierte Zugriffstoken ein. Es wird nicht im Browser gespeichert.',
      tokenLabel: 'Zugriffstoken', connect: 'Verbinden', configuration: 'KONFIGURATION', accountsTitle: 'Synthetische Konten',
      accountsText: 'Testdaten werden im Speicher der Anwendung oder des Dienstes abgelegt.',
      emptyTitle: 'Noch keine Testkonten', emptyText: 'Legen Sie unten ein synthetisches Konto an, um das Speichern und erneute Laden auszuprobieren.',
      addTitle: 'Testkonto hinzufügen', accountName: 'Anzeigename', accountAddress: 'Testadresse', protocol: 'Protokoll',
      namePlaceholder: 'Beispielkonto', addressPlaceholder: 'beispiel@example.invalid',
      noCredentials: 'Es werden keine Zugangsdaten abgefragt.', saveAccount: 'Testkonto speichern',
      availability: 'VERFÜGBARKEIT', roadmapTitle: 'Was als Nächstes kommt',
      roadmapText: 'Diese Funktionen gehören zu späteren Ausbaustufen und sind noch nicht aktiv.',
      roadmapMail: 'E-Mail & Konten', roadmapMailDetail: 'IMAP, SMTP und JMAP', roadmapCalendar: 'Kalender & Kontakte',
      roadmapCalendarDetail: 'CalDAV und CardDAV', roadmapSync: 'Offline & Abgleich', roadmapSyncDetail: 'Cache und Warteschlange',
      unavailable: 'Nicht verfügbar', foundationNote: 'Ihre Daten bleiben bei Ihnen. Der gemeinsame Kern benötigt kein Herstellerkonto und keinen Cloud-Dienst.',
      disconnect: 'Verbindung trennen', retry: 'Erneut versuchen', desktop: 'Lokale Anwendung', web: 'Eigener Webdienst',
      ready: 'Testdaten geladen.', saved: 'Testkonto gespeichert.', disconnected: 'Verbindung getrennt.',
      saveRefreshFailed: 'Das Konto wurde gespeichert, aber die Anzeige konnte nicht aktualisiert werden.',
      error: 'Der Vorgang ist fehlgeschlagen.', AUTH_REQUIRED: 'Das Zugriffstoken ist ungültig.',
      LOCAL_AUTH_REQUIRED: 'Die lokale Sitzung ist abgelaufen. Starten Sie die Anwendung neu und verwenden Sie deren frisch geöffnete Browserseite.',
      CONNECTION_FAILED: 'Der lokale Dienst ist nicht erreichbar.', STORE_LOCKED: 'Der Speicher wird gerade von einem anderen Prozess verwendet.',
      UNSUPPORTED_SCHEMA: 'Die Datenversion wird nicht unterstützt; vorhandene Daten wurden nicht verändert.',
      INVALID_STORED_DATA: 'Die gespeicherten Daten sind ungültig; vorhandene Daten wurden nicht verändert.',
      INVALID_ACCOUNT: 'Bitte geben Sie einen Anzeigenamen und eine gültige Testadresse ein.',
      INVALID_DATA: 'Bitte überprüfen Sie die eingegebenen Werte.',
      READ_FAILED: 'Der Anwendungsspeicher konnte nicht gelesen werden.',
      WRITE_FAILED: 'Der Anwendungsspeicher konnte nicht geschrieben werden; der letzte Stand bleibt erhalten.',
      STORAGE_UNAVAILABLE: 'Der Anwendungsspeicher ist nicht verfügbar.',
      COMMIT_UNCERTAIN: 'Der Vorgang wurde möglicherweise gespeichert, aber die Sperre konnte nicht freigegeben werden. Bitte den Datenstand prüfen.',
      LOCK_CLEANUP_FAILED: 'Die Speichersperre konnte nicht freigegeben werden. Bitte die Wiederherstellungshinweise beachten.',
      ORIGIN_REJECTED: 'Der Zugriff wurde wegen einer ungültigen Adresse abgelehnt.',
    },
    en: {
      skip: 'Skip to content', workspace: 'WORKSPACE', workspaceName: 'Workspace',
      overview: 'Overview', inbox: 'Inbox', calendar: 'Calendar', tasks: 'Tasks', contacts: 'Contacts', later: 'LATER',
      privateTitle: 'Private from the start', privateDetail: 'No telemetry. No cloud required.',
      foundation: 'THE FOUNDATION', headline: 'Welcome to Atlas Mail.',
      intro: 'A private place for your communication, built to be open and local first.',
      noticeTitle: 'This foundation is not yet a mail client',
      noticeText: 'Only synthetic test accounts can be saved here. Email, synchronization, and calendars are not available yet. Do not enter real credentials.',
      webAccess: 'WEB ACCESS', loginTitle: 'Connect to your own service',
      loginText: 'Enter the access token configured at startup. It is not stored in your browser.',
      tokenLabel: 'Access token', connect: 'Connect', configuration: 'CONFIGURATION', accountsTitle: 'Synthetic accounts',
      accountsText: 'Test data is stored by the application or its service.',
      emptyTitle: 'No test accounts yet', emptyText: 'Add a synthetic account below to try saving and reloading it.',
      addTitle: 'Add a test account', accountName: 'Display name', accountAddress: 'Test address', protocol: 'Protocol',
      namePlaceholder: 'Example account', addressPlaceholder: 'example@example.invalid',
      noCredentials: 'No credentials are requested.', saveAccount: 'Save test account',
      availability: 'AVAILABILITY', roadmapTitle: 'What comes next',
      roadmapText: 'These features belong to later stages and are not active yet.',
      roadmapMail: 'Email & accounts', roadmapMailDetail: 'IMAP, SMTP and JMAP', roadmapCalendar: 'Calendar & contacts',
      roadmapCalendarDetail: 'CalDAV and CardDAV', roadmapSync: 'Offline & sync', roadmapSyncDetail: 'Cache and queue',
      unavailable: 'Unavailable', foundationNote: 'Your data stays with you. The shared core does not require a vendor account or cloud service.',
      disconnect: 'Disconnect', retry: 'Try again', desktop: 'Local application', web: 'Your web service',
      ready: 'Test data loaded.', saved: 'Test account saved.', disconnected: 'Disconnected.',
      saveRefreshFailed: 'The account was saved, but the display could not be refreshed.',
      error: 'The operation failed.', AUTH_REQUIRED: 'The access token is invalid.',
      LOCAL_AUTH_REQUIRED: 'The local session has expired. Restart the application and use its newly opened browser page.',
      CONNECTION_FAILED: 'The local service is unavailable.', STORE_LOCKED: 'The store is in use by another process.',
      UNSUPPORTED_SCHEMA: 'The data version is unsupported; existing data was not changed.',
      INVALID_STORED_DATA: 'The stored data is invalid; existing data was not changed.',
      INVALID_ACCOUNT: 'Enter a display name and a valid test address.',
      INVALID_DATA: 'Please check the entered values.',
      READ_FAILED: 'The application store could not be read.',
      WRITE_FAILED: 'The application store could not be written; the previous state remains intact.',
      STORAGE_UNAVAILABLE: 'The application store is unavailable.',
      COMMIT_UNCERTAIN: 'The operation may have been saved, but its lock could not be released. Check the stored data.',
      LOCK_CLEANUP_FAILED: 'The storage lock could not be released. Consult the recovery instructions.',
      ORIGIN_REJECTED: 'The request was rejected because its origin is invalid.',
    },
  };
  const $ = (id) => document.getElementById(id);
  const desktop = window.location.pathname === '/desktop';
  let language = navigator.language.toLowerCase().startsWith('de') ? 'de' : 'en';
  let token = '';
  let connected = desktop;
  let connectionVersion = 0;
  let state = null;
  let statusKey = '';
  let errorCode = '';
  let busy = false;

  function translate(key) {
    return messages[language][key] || key;
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
    const visibleError = desktop && errorCode === 'AUTH_REQUIRED' ? 'LOCAL_AUTH_REQUIRED' : errorCode;
    $('error').textContent = errorCode ? `${translate(visibleError in messages[language] ? visibleError : 'error')} (${errorCode})` : '';
  }

  function renderState() {
    const accounts = state ? state.accounts : [];
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
      information.append(name, address);
      const protocol = document.createElement('span');
      protocol.className = 'protocol-tag';
      protocol.textContent = account.protocol === 'jmap' ? 'JMAP' : 'IMAP / SMTP';
      item.append(avatar, information, protocol);
      list.append(item);
    }
    $('login-section').hidden = desktop || connected;
    $('disconnect').hidden = desktop || !connected;
    $('dashboard').hidden = !connected || !state;
    $('retry').hidden = !(connected && errorCode);
    for (const control of $('account-form').elements) {
      control.disabled = !connected || !state || busy;
    }
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
      if (operation === 'addAccount') {
        endpoint = '/api/v1/accounts';
        options.method = 'POST';
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(input);
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
          headers: { Authorization: 'Bearer ' + launchToken },
          cache: 'no-store',
        });
        launchToken = '';
        if (!response.ok) {
          setStatus('', 'AUTH_REQUIRED');
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
    renderState();
    return true;
  }

  $('language-de').addEventListener('click', () => { language = 'de'; renderLanguage(); });
  $('language-en').addEventListener('click', () => { language = 'en'; renderLanguage(); });
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
    setStatus('disconnected');
    renderState();
  });
  $('retry').addEventListener('click', () => refresh());
  $('account-form').addEventListener('submit', async (event) => {
    event.preventDefault();
    if (busy || !connected) return;
    busy = true;
    renderState();
    const version = connectionVersion;
    const result = await request('addAccount', {
      label: $('account-name').value,
      address: $('account-address').value,
      protocol: $('account-protocol').value,
    });
    if (version !== connectionVersion) {
      busy = false;
      renderState();
      return;
    }
    if (!result.ok) {
      setStatus('', result.code || 'UNEXPECTED_ERROR');
    } else {
      $('account-form').reset();
      if (!(await refresh('saved', version)) && version === connectionVersion) {
        setStatus('', 'saveRefreshFailed');
      }
    }
    busy = false;
    renderState();
  });

  renderLanguage();
  renderState();
  if (desktop) connectLocalSession();
})();
