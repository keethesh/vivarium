// Hexagon Application

document.addEventListener('DOMContentLoaded', () => {
    // State
    const state = {
        connected: false,
        activeAttacks: {},
        stats: {
            requests: 0,
            successful: 0,
            failed: 0,
            rps: 0,
        },
    };

    // DOM Elements
    const elements = {
        connectionStatus: document.getElementById('connectionStatus'),
        target: document.getElementById('target'),
        attackType: document.getElementById('attackType'),
        launchBtn: document.getElementById('launchBtn'),
        stopBtn: document.getElementById('stopBtn'),
        logConsole: document.getElementById('logConsole'),
        attacksList: document.getElementById('attacksList'),
        // Stats
        statRequests: document.getElementById('statRequests'),
        statSuccess: document.getElementById('statSuccess'),
        statFailed: document.getElementById('statFailed'),
        statRPS: document.getElementById('statRPS'),
        progressFill: document.getElementById('progressFill'),
        progressText: document.getElementById('progressText'),
        // Attack options
        locustOptions: document.getElementById('locustOptions'),
        tickOptions: document.getElementById('tickOptions'),
        flyswarmOptions: document.getElementById('flyswarmOptions'),
    };

    // Initialize WebSocket
    WS.on('connected', () => {
        state.connected = true;
        updateConnectionStatus(true);
        log('Connected to Hexagon server', 'success');
    });

    WS.on('disconnected', () => {
        state.connected = false;
        updateConnectionStatus(false);
        log('Disconnected from server', 'error');
    });

    WS.on('log', (data) => {
        log(data.message, 'info');
    });

    WS.on('progress', (data) => {
        updateStats(data);
        updateProgress(data);
    });

    WS.on('complete', (data) => {
        log(`✓ Attack ${data.attackId} completed - ${data.totalRequests || data.packetsSent || data.connections} total`, 'success');
        removeActiveAttack(data.attackId);
        elements.progressText.textContent = 'Completed';
    });

    WS.on('error', (data) => {
        log(`✗ Attack error: ${data.error}`, 'error');
        if (data.attackId) {
            removeActiveAttack(data.attackId);
        }
    });

    WS.connect();

    // Event Listeners
    elements.attackType.addEventListener('change', updateAttackOptions);
    elements.launchBtn.addEventListener('click', launchAttack);
    elements.stopBtn.addEventListener('click', stopAllAttacks);

    // Update attack options visibility
    function updateAttackOptions() {
        const type = elements.attackType.value;

        elements.locustOptions.classList.add('hidden');
        elements.tickOptions.classList.add('hidden');
        elements.flyswarmOptions.classList.add('hidden');

        if (type === 'locust') {
            elements.locustOptions.classList.remove('hidden');
        } else if (type === 'tick') {
            elements.tickOptions.classList.remove('hidden');
        } else if (type === 'flyswarm') {
            elements.flyswarmOptions.classList.remove('hidden');
        }
    }

    // Launch attack
    async function launchAttack() {
        const target = elements.target.value.trim();
        if (!target) {
            log('Please enter a target URL', 'error');
            return;
        }

        const type = elements.attackType.value;
        elements.launchBtn.disabled = true;

        // Reset stats
        resetStats();

        try {
            let result;

            switch (type) {
                case 'locust':
                    result = await API.launchLocust(
                        target,
                        document.getElementById('locustRounds').value,
                        document.getElementById('locustConcurrency').value
                    );
                    break;
                case 'tick':
                    result = await API.launchTick(
                        target,
                        document.getElementById('tickSockets').value,
                        document.getElementById('tickDelay').value
                    );
                    break;
                case 'flyswarm':
                    result = await API.launchFlySwarm(
                        target,
                        document.getElementById('flyswarmRounds').value,
                        document.getElementById('flyswarmPort').value,
                        document.getElementById('flyswarmConcurrency').value,
                        document.getElementById('flyswarmPacketSize').value
                    );
                    break;
            }

            if (result.status === 'started') {
                addActiveAttack(result.attackId, result.type, result.target);
                log(`Launched ${result.type} attack on ${result.target}`, 'info');
            }
        } catch (error) {
            log(`Failed to launch attack: ${error.message}`, 'error');
        } finally {
            elements.launchBtn.disabled = false;
        }
    }

    // Stop all attacks
    async function stopAllAttacks() {
        try {
            await API.stopAttack();
            log('Stopping all attacks...', 'warning');
            state.activeAttacks = {};
            updateAttacksList();
        } catch (error) {
            log(`Failed to stop attacks: ${error.message}`, 'error');
        }
    }

    // Update connection status
    function updateConnectionStatus(connected) {
        const statusEl = elements.connectionStatus;
        const textEl = statusEl.querySelector('.status-text');

        if (connected) {
            statusEl.classList.add('connected');
            statusEl.classList.remove('disconnected');
            textEl.textContent = 'Connected';
        } else {
            statusEl.classList.remove('connected');
            statusEl.classList.add('disconnected');
            textEl.textContent = 'Disconnected';
        }
    }

    // Update stats display
    function updateStats(data) {
        state.stats.requests = data.completed || 0;
        state.stats.successful = data.successful || 0;
        state.stats.failed = data.failed || 0;
        state.stats.rps = data.rps || 0;

        elements.statRequests.textContent = formatNumber(state.stats.requests);
        elements.statSuccess.textContent = formatNumber(state.stats.successful);
        elements.statFailed.textContent = formatNumber(state.stats.failed);
        elements.statRPS.textContent = state.stats.rps.toFixed(1);
    }

    // Update progress bar
    function updateProgress(data) {
        if (data.total > 0) {
            const percent = (data.completed / data.total) * 100;
            elements.progressFill.style.width = percent + '%';
            elements.progressText.textContent = `${formatNumber(data.completed)} / ${formatNumber(data.total)} (${percent.toFixed(1)}%)`;
        }
    }

    // Reset stats
    function resetStats() {
        state.stats = { requests: 0, successful: 0, failed: 0, rps: 0 };
        elements.statRequests.textContent = '0';
        elements.statSuccess.textContent = '0';
        elements.statFailed.textContent = '0';
        elements.statRPS.textContent = '0';
        elements.progressFill.style.width = '0%';
        elements.progressText.textContent = 'Starting...';
    }

    // Add active attack
    function addActiveAttack(id, type, target) {
        state.activeAttacks[id] = { type, target };
        updateAttacksList();
    }

    // Remove active attack
    function removeActiveAttack(id) {
        delete state.activeAttacks[id];
        updateAttacksList();
    }

    // Update attacks list
    function updateAttacksList() {
        const ids = Object.keys(state.activeAttacks);

        if (ids.length === 0) {
            elements.attacksList.innerHTML = '<div class="empty-state">No active attacks</div>';
            return;
        }

        elements.attacksList.innerHTML = ids.map(id => {
            const attack = state.activeAttacks[id];
            return `
                <div class="attack-item" data-id="${id}">
                    <div class="attack-info">
                        <span class="attack-type">${attack.type.toUpperCase()}</span>
                        <span>${truncate(attack.target, 40)}</span>
                    </div>
                    <button class="stop-btn" onclick="stopSingleAttack('${id}')">Stop</button>
                </div>
            `;
        }).join('');
    }

    // Log message
    function log(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const entry = document.createElement('div');
        entry.className = `log-entry ${type}`;
        entry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span> ${message}`;

        elements.logConsole.appendChild(entry);
        elements.logConsole.scrollTop = elements.logConsole.scrollHeight;

        // Limit log entries
        while (elements.logConsole.children.length > 100) {
            elements.logConsole.removeChild(elements.logConsole.firstChild);
        }
    }

    // Helpers
    function formatNumber(num) {
        return num.toLocaleString();
    }

    function truncate(str, len) {
        return str.length > len ? str.substring(0, len - 3) + '...' : str;
    }

    // Global function for stopping single attack
    window.stopSingleAttack = async function (id) {
        try {
            await API.stopAttack(id);
            log(`Stopping attack ${id}...`, 'warning');
        } catch (error) {
            log(`Failed to stop attack: ${error.message}`, 'error');
        }
    };

    // Initial setup
    updateAttackOptions();
});
