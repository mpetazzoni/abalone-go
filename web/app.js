// Abalone Game Client
// =====================
// SVG hex board rendering, WebSocket client, and game interaction.

(function () {
    'use strict';

    // ---------------------
    // Constants
    // ---------------------

    const HEX_SIZE = 28;
    const BOARD_RADIUS = 4;
    const SVG_NS = 'http://www.w3.org/2000/svg';

    // Six hex directions in axial coordinates
    // Angles match flat-top hex layout pixel positions from hexToPixel()
    const DIRECTIONS = [
        { dq: 1, dr: 0, name: 'E', angle: 30 },
        { dq: 1, dr: -1, name: 'NE', angle: -30 },
        { dq: 0, dr: -1, name: 'NW', angle: -90 },
        { dq: -1, dr: 0, name: 'W', angle: 210 },
        { dq: -1, dr: 1, name: 'SW', angle: 150 },
        { dq: 0, dr: 1, name: 'SE', angle: 90 },
    ];

    // Colors
    const COLOR_BOARD_BG = '#2d5016';
    const COLOR_CELL = '#3a6b1e';
    const COLOR_CELL_STROKE = '#2d5016';
    const COLOR_SELECTED = '#ffd700';
    const COLOR_ARROW = '#ffa500';
    const COLOR_ARROW_HOVER = '#ffcc00';

    // ---------------------
    // State
    // ---------------------

    let ws = null;
    let myColor = null;       // 'black' or 'white'
    let gameState = null;     // latest state message from server
    let selectedMarbles = []; // array of {q, r}
    let gameCode = null;
    let gameOver = false;
    let prevCaptured = { black: 0, white: 0 }; // previous captured counts for pulse detection
    let shiftHeld = false;
    // (animation state is computed locally inside renderBoard)

    // SVG dimensions
    const SVG_WIDTH = 580;
    const SVG_HEIGHT = 540;
    const CENTER_X = SVG_WIDTH / 2;
    const CENTER_Y = SVG_HEIGHT / 2;

    // ---------------------
    // DOM references (set on init)
    // ---------------------

    let screenLobby, screenWaiting, screenGame, screenRules;
    let codeValue;
    let btnHost, btnJoin, inputCode;
    let svgBoard;
    let boardDynamic;
    let statusTurn, statusColor, statusCaptured, statusMessage;
    let gameOverOverlay, gameOverText, btnPlayAgain;

    // ---------------------
    // Initialization
    // ---------------------

    document.addEventListener('DOMContentLoaded', init);

    function init() {
        // Cache DOM references
        screenLobby = document.getElementById('screen-lobby');
        screenWaiting = document.getElementById('screen-waiting');
        screenGame = document.getElementById('screen-game');
        screenRules = document.getElementById('screen-rules');

        codeValue = document.getElementById('code-value');

        btnHost = document.getElementById('btn-host');
        btnJoin = document.getElementById('btn-join');
        inputCode = document.getElementById('input-code');

        svgBoard = document.getElementById('svg-board');

        statusTurn = document.getElementById('status-turn');
        statusColor = document.getElementById('status-color');
        statusCaptured = document.getElementById('status-captured');
        statusMessage = document.getElementById('status-message');

        gameOverOverlay = document.getElementById('game-over-overlay');
        gameOverText = document.getElementById('game-over-text');
        btnPlayAgain = document.getElementById('btn-play-again');

        // Event listeners
        btnHost.addEventListener('click', hostGame);
        btnJoin.addEventListener('click', joinGame);
        inputCode.addEventListener('keydown', function (e) {
            if (e.key === 'Enter') joinGame();
        });
        btnPlayAgain.addEventListener('click', returnToLobby);

        document.getElementById('btn-rules').addEventListener('click', function () {
            screenRules.classList.toggle('hidden');
        });
        document.getElementById('btn-rules-back').addEventListener('click', function () {
            screenRules.classList.add('hidden');
        });

        // Copy code button
        document.getElementById('btn-copy-code').addEventListener('click', function () {
            if (gameCode) {
                navigator.clipboard.writeText(gameCode).then(function () {
                    const btn = document.getElementById('btn-copy-code');
                    btn.textContent = 'Copied!';
                    setTimeout(function () { btn.textContent = 'Copy'; }, 1500);
                });
            }
        });

        // Track Shift key for multi-select
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Shift' && !shiftHeld) { shiftHeld = true; renderBoard(); }
        });
        document.addEventListener('keyup', function(e) {
            if (e.key === 'Shift') { shiftHeld = false; renderBoard(); }
        });
        window.addEventListener('blur', function() {
            if (shiftHeld) { shiftHeld = false; renderBoard(); }
        });

        setupBoard();
        showScreen('lobby');
    }

    function setupBoard() {
        // Static defs: gradients and filters (created once, never change)
        const defs = createSVG('defs');

        // Black marble gradient
        const blackGrad = createSVG('radialGradient', {
            id: 'grad-black', cx: '50%', cy: '50%', r: '50%', fx: '35%', fy: '30%',
        });
        blackGrad.appendChild(createSVG('stop', { offset: '0%', 'stop-color': '#777' }));
        blackGrad.appendChild(createSVG('stop', { offset: '50%', 'stop-color': '#333' }));
        blackGrad.appendChild(createSVG('stop', { offset: '100%', 'stop-color': '#111' }));
        defs.appendChild(blackGrad);

        // White marble gradient
        const whiteGrad = createSVG('radialGradient', {
            id: 'grad-white', cx: '50%', cy: '50%', r: '50%', fx: '35%', fy: '30%',
        });
        whiteGrad.appendChild(createSVG('stop', { offset: '0%', 'stop-color': '#fff' }));
        whiteGrad.appendChild(createSVG('stop', { offset: '40%', 'stop-color': '#eee' }));
        whiteGrad.appendChild(createSVG('stop', { offset: '100%', 'stop-color': '#bbb' }));
        defs.appendChild(whiteGrad);

        // Board glow filter
        const glowFilter = createSVG('filter', { id: 'board-glow', x: '-20%', y: '-20%', width: '140%', height: '140%' });
        glowFilter.appendChild(createSVG('feGaussianBlur', { 'in': 'SourceGraphic', stdDeviation: '8' }));
        defs.appendChild(glowFilter);

        svgBoard.appendChild(defs);

        // Static board background
        renderBoardBackground();

        // Dynamic layer for cells, marbles, arrows
        boardDynamic = createSVG('g', { 'class': 'board-dynamic' });
        svgBoard.appendChild(boardDynamic);
    }

    // ---------------------
    // Screen management
    // ---------------------

    function showScreen(name) {
        const screens = [screenLobby, screenWaiting, screenGame];
        let target = null;

        switch (name) {
            case 'lobby': target = screenLobby; break;
            case 'waiting': target = screenWaiting; break;
            case 'game': target = screenGame; break;
        }

        screens.forEach(function(s) {
            if (s !== target) {
                s.classList.add('hidden');
                s.classList.remove('screen-visible');
            }
        });

        if (target) {
            target.classList.remove('hidden');
            // Trigger fade-in on next frame
            requestAnimationFrame(function() {
                target.classList.add('screen-visible');
            });
        }
    }

    // ---------------------
    // Lobby actions
    // ---------------------

    function hostGame() {
        connect('create');
    }

    function joinGame() {
        const code = inputCode.value.trim().toLowerCase();
        if (!code) {
            showStatusMessage('Please enter a game code.', true);
            return;
        }
        connect('join', code);
    }

    function returnToLobby() {
        if (ws) {
            ws.close();
            ws = null;
        }
        myColor = null;
        gameState = null;
        selectedMarbles = [];
        gameCode = null;
        gameOver = false;
        prevCaptured = { black: 0, white: 0 };
        gameOverOverlay.classList.add('hidden');
        showScreen('lobby');
    }

    // ---------------------
    // WebSocket
    // ---------------------

    function connect(action, code) {
        if (ws) {
            ws.close();
            ws = null;
        }

        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        let url = protocol + '//' + location.host + '/ws?action=' + action;
        if (code) {
            url += '&code=' + encodeURIComponent(code);
        }

        ws = new WebSocket(url);

        ws.onmessage = function (event) {
            handleMessage(JSON.parse(event.data));
        };

        ws.onclose = function () {
            if (!gameOver) {
                showStatusMessage('Connection lost.', true);
            }
        };

        ws.onerror = function () {
            showStatusMessage('Connection error.', true);
        };
    }

    function handleMessage(msg) {
        switch (msg.type) {
            case 'waiting':
                gameCode = msg.code;
                myColor = msg.color;
                codeValue.textContent = gameCode;
                showScreen('waiting');
                break;

            case 'joined':
                myColor = msg.color;
                gameOver = false;
                gameOverOverlay.classList.add('hidden');
                showScreen('game');
                break;

            case 'state': {
                const oldBoard = gameState ? Object.assign({}, gameState.board) : null;
                gameState = msg;
                selectedMarbles = [];
                renderBoard(oldBoard);
                updateStatus();
                const statusBar = document.querySelector('.status-bar');
                statusBar.classList.remove('turn-flash');
                void statusBar.offsetWidth; // force reflow
                statusBar.classList.add('turn-flash');
                break;
            }

            case 'game_over': {
                const oldBoard = gameState ? Object.assign({}, gameState.board) : null;
                gameState = msg;
                gameOver = true;
                selectedMarbles = [];
                renderBoard(oldBoard);
                updateStatus();
                showGameOver(msg.winner);
                break;
            }

            case 'opponent_disconnected':
                gameOver = true;
                showStatusMessage(msg.message || 'Your opponent disconnected.', true);
                gameOverText.textContent = 'Opponent Disconnected';
                gameOverOverlay.className = 'game-over-overlay';
                gameOverOverlay.classList.remove('hidden');
                break;

            case 'error':
                showStatusMessage(msg.error || msg.message || 'Unknown error', true);
                break;
        }
    }

    // ---------------------
    // Status display
    // ---------------------

    function updateStatus() {
        if (!gameState) return;

        const colorIcon = myColor === 'black' ? '⚫' : '⚪';
        statusColor.textContent = 'You are ' + colorIcon + ' ' + capitalize(myColor);

        const isMyTurn = gameState.turn === myColor;
        statusTurn.textContent = isMyTurn ? 'Your turn' : 'Opponent\'s turn';
        statusTurn.className = 'status-turn' + (isMyTurn ? ' my-turn' : '');

        const capB = gameState.captured ? gameState.captured.black : 0;
        const capW = gameState.captured ? gameState.captured.white : 0;

        const newCapBlack = capB > prevCaptured.black;
        const newCapWhite = capW > prevCaptured.white;
        prevCaptured = { black: capB, white: capW };

        statusCaptured.innerHTML = buildCapturedTray(capB, capW, newCapBlack, newCapWhite);
        statusMessage.textContent = '';
    }

    function buildCapturedTray(capBlack, capWhite, pulseBlack, pulseWhite) {
        let html = '<span class="cap-tray">';
        // Black captured (by white)
        for (let i = 0; i < 6; i++) {
            const filled = i < capBlack;
            const justCaptured = filled && pulseBlack && i === capBlack - 1;
            html += '<span class="cap-marble cap-marble-black' + (filled ? ' filled' : '') + (justCaptured ? ' just-captured' : '') + '"></span>';
        }
        html += '<span class="cap-sep">|</span>';
        // White captured (by black)
        for (let i = 0; i < 6; i++) {
            const filled = i < capWhite;
            const justCaptured = filled && pulseWhite && i === capWhite - 1;
            html += '<span class="cap-marble cap-marble-white' + (filled ? ' filled' : '') + (justCaptured ? ' just-captured' : '') + '"></span>';
        }
        html += '</span>';
        return html;
    }

    function showStatusMessage(text, isError) {
        if (statusMessage) {
            statusMessage.textContent = text;
            statusMessage.className = 'status-message' + (isError ? ' error' : '');
            if (!isError) {
                setTimeout(function () { statusMessage.textContent = ''; }, 3000);
            }
        }
    }

    function showGameOver(winner) {
        const isWinner = winner === myColor;
        gameOverText.textContent = isWinner ? 'You win!' : capitalize(winner) + ' wins!';
        gameOverOverlay.className = 'game-over-overlay ' + (isWinner ? 'win' : 'lose');
        gameOverOverlay.classList.remove('hidden');
    }

    function capitalize(s) {
        return s ? s.charAt(0).toUpperCase() + s.slice(1) : '';
    }

    // ---------------------
    // Hex math
    // ---------------------

    function hexToPixel(q, r) {
        const x = HEX_SIZE * (3 / 2 * q);
        const y = HEX_SIZE * (Math.sqrt(3) / 2 * q + Math.sqrt(3) * r);
        return { x: x + CENTER_X, y: y + CENTER_Y };
    }

    function hexKey(q, r) {
        return q + ',' + r;
    }

    function isValidHex(q, r) {
        return Math.abs(q) <= BOARD_RADIUS &&
            Math.abs(r) <= BOARD_RADIUS &&
            Math.abs(q + r) <= BOARD_RADIUS;
    }

    function getCellValue(q, r) {
        if (!gameState || !gameState.board) return 0;
        return gameState.board[hexKey(q, r)] || 0;
    }

    // ---------------------
    // Selection logic
    // ---------------------

    function isSelected(q, r) {
        return selectedMarbles.some(function (m) {
            return m.q === q && m.r === r;
        });
    }

    function toggleSelection(q, r) {
        if (!gameState || gameState.turn !== myColor || gameOver) return;

        const val = getCellValue(q, r);
        const myVal = myColor === 'black' ? 1 : 2;
        if (val !== myVal) {
            selectedMarbles = [];
            renderBoard();
            return;
        }

        if (!shiftHeld) {
            if (isSelected(q, r) && selectedMarbles.length === 1) {
                selectedMarbles = [];
            } else {
                selectedMarbles = [{ q: q, r: r }];
            }
        } else {
            if (isSelected(q, r)) {
                selectedMarbles = selectedMarbles.filter(function(m) {
                    return m.q !== q || m.r !== r;
                });
            } else {
                const next = selectedMarbles.concat([{ q: q, r: r }]);
                if (next.length === 1) {
                    selectedMarbles = next;
                } else if (next.length === 2) {
                    selectedMarbles = areAdjacent(next[0], next[1]) ? next : [{ q: q, r: r }];
                } else if (next.length === 3) {
                    selectedMarbles = areCollinear(next) ? sortMarbles(next) : [{ q: q, r: r }];
                } else {
                    selectedMarbles = [{ q: q, r: r }];
                }
            }
        }

        renderBoard();
    }

    function areAdjacent(a, b) {
        const dq = b.q - a.q;
        const dr = b.r - a.r;
        return DIRECTIONS.some(function (d) {
            return d.dq === dq && d.dr === dr;
        });
    }

    function areCollinear(marbles) {
        if (marbles.length <= 1) return true;
        if (marbles.length === 2) return areAdjacent(marbles[0], marbles[1]);

        // For 3 marbles: sort them and check they form a line
        const sorted = sortMarbles(marbles);
        const dq1 = sorted[1].q - sorted[0].q;
        const dr1 = sorted[1].r - sorted[0].r;
        const dq2 = sorted[2].q - sorted[1].q;
        const dr2 = sorted[2].r - sorted[1].r;

        // Must be same direction and adjacent
        if (dq1 !== dq2 || dr1 !== dr2) return false;
        return DIRECTIONS.some(function (d) {
            return d.dq === dq1 && d.dr === dr1;
        });
    }

    function sortMarbles(marbles) {
        return marbles.slice().sort(function (a, b) {
            return a.q !== b.q ? a.q - b.q : a.r - b.r;
        });
    }

    // ---------------------
    // Move submission
    // ---------------------

    function sendMove(dq, dr) {
        if (!ws || selectedMarbles.length === 0) return;

        const msg = {
            type: 'move',
            marbles: selectedMarbles.map(function (m) { return [m.q, m.r]; }),
            direction: [dq, dr]
        };

        ws.send(JSON.stringify(msg));
        selectedMarbles = [];
        renderBoard();
    }

    // ---------------------
    // SVG Board Rendering
    // ---------------------

    function renderBoard(oldBoard) {
        // Clear dynamic layer only (defs and background are static)
        while (boardDynamic.firstChild) {
            boardDynamic.removeChild(boardDynamic.firstChild);
        }

        // Render cells
        for (let q = -BOARD_RADIUS; q <= BOARD_RADIUS; q++) {
            for (let r = -BOARD_RADIUS; r <= BOARD_RADIUS; r++) {
                if (!isValidHex(q, r)) continue;
                renderCell(q, r);
            }
        }

        // Compute move direction from board diff (for animation)
        let moveDir = null;
        const newPositions = (gameState && gameState.board) ? gameState.board : {};

        if (oldBoard) {
            moveDir = detectMoveDirection(oldBoard, newPositions);
        }

        // Render marbles with animation
        if (gameState && gameState.board) {
            for (const key in newPositions) {
                const parts = key.split(',');
                const mq = parseInt(parts[0], 10);
                const mr = parseInt(parts[1], 10);
                const val = newPositions[key];

                if (oldBoard) {
                    if (oldBoard[key] === val) {
                        renderMarble(mq, mr, val);
                        continue;
                    }
                    if (moveDir) {
                        renderMarbleAnimated(mq, mr, val, mq - moveDir.dq, mr - moveDir.dr);
                        continue;
                    }
                }

                renderMarble(mq, mr, val);
            }

            // Render push-off animations for marbles that left the board
            if (oldBoard && moveDir) {
                for (const oldKey in oldBoard) {
                    const oldVal = oldBoard[oldKey];
                    if (newPositions[oldKey] === oldVal) continue;
                    let wasMoved = false;
                    for (const newKey in newPositions) {
                        if (newPositions[newKey] === oldVal && oldBoard[newKey] !== oldVal) {
                            const np = newKey.split(',');
                            const srcKey = (parseInt(np[0], 10) - moveDir.dq) + ',' + (parseInt(np[1], 10) - moveDir.dr);
                            if (srcKey === oldKey) {
                                wasMoved = true;
                                break;
                            }
                        }
                    }
                    if (wasMoved) continue;
                    const oParts = oldKey.split(',');
                    const oq = parseInt(oParts[0], 10);
                    const or_ = parseInt(oParts[1], 10);
                    const offPixel = hexToPixel(oq + moveDir.dq, or_ + moveDir.dr);
                    renderMarblePushOff(oq, or_, offPixel.x, offPixel.y, oldVal);
                }
            }
        }

        // Render direction arrows near selection
        renderDirectionArrows();
    }

    // Detect the uniform move direction by comparing old and new board states.
    // Returns {dq, dr} or null if no direction can be determined.
    function detectMoveDirection(oldBoard, newPositions) {
        // Collect new and vacated positions for each color
        for (let ci = 1; ci <= 2; ci++) {
            const newPos = [];
            const vacated = {};
            let vacatedCount = 0;

            for (const key in newPositions) {
                if (newPositions[key] === ci && oldBoard[key] !== ci) {
                    const parts = key.split(',');
                    newPos.push({ q: parseInt(parts[0], 10), r: parseInt(parts[1], 10) });
                }
            }
            for (const key in oldBoard) {
                if (oldBoard[key] === ci && newPositions[key] !== ci) {
                    vacated[key] = true;
                    vacatedCount++;
                }
            }

            if (newPos.length === 0) continue;

            // Try each of the 6 directions
            for (let di = 0; di < DIRECTIONS.length; di++) {
                const dir = DIRECTIONS[di];
                let match = true;
                for (let ni = 0; ni < newPos.length; ni++) {
                    const srcKey = (newPos[ni].q - dir.dq) + ',' + (newPos[ni].r - dir.dr);
                    if (!vacated[srcKey]) {
                        match = false;
                        break;
                    }
                }
                if (match && newPos.length === vacatedCount) {
                    return { dq: dir.dq, dr: dir.dr };
                }
            }
        }
        return null;
    }

    function renderBoardBackground() {
        // Draw a large hexagonal background
        const bgSize = HEX_SIZE * (BOARD_RADIUS + 0.8) * Math.sqrt(3);
        const corners = [];
        for (let i = 0; i < 6; i++) {
            const angle = Math.PI / 180 * (60 * i + 30);
            corners.push(CENTER_X + bgSize * Math.cos(angle) + ',' + (CENTER_Y + bgSize * Math.sin(angle)));
        }

        // Glow layer behind the board
        const glowBg = createSVG('polygon', {
            points: corners.join(' '),
            fill: '#3a7a22',
            opacity: '0.15',
            filter: 'url(#board-glow)',
        });
        svgBoard.appendChild(glowBg);

        const bg = createSVG('polygon', {
            points: corners.join(' '),
            fill: COLOR_BOARD_BG,
            stroke: '#1a3a0a',
            'stroke-width': '3'
        });
        svgBoard.appendChild(bg);
    }

    function renderCell(q, r) {
        const pos = hexToPixel(q, r);
        const cellSize = HEX_SIZE * 0.88;
        const cellR = cellSize * 0.62;

        // Outer rim (lighter edge for 3D pit effect)
        const rim = createSVG('circle', {
            cx: pos.x, cy: pos.y - 1, r: cellR,
            fill: '#4a8a2e',
            opacity: '0.5',
        });
        boardDynamic.appendChild(rim);

        // Main pit
        const cell = createSVG('circle', {
            cx: pos.x,
            cy: pos.y,
            r: cellR,
            fill: COLOR_CELL,
            stroke: COLOR_CELL_STROKE,
            'stroke-width': '1',
            'class': 'cell',
            'data-q': q,
            'data-r': r,
        });

        cell.addEventListener('click', function () {
            toggleSelection(q, r);
        });

        boardDynamic.appendChild(cell);

        // Inner shadow (darker center for depth)
        const innerShadow = createSVG('circle', {
            cx: pos.x, cy: pos.y + 1, r: cellR * 0.85,
            fill: '#2a5014',
            opacity: '0.4',
            'pointer-events': 'none',
        });
        boardDynamic.appendChild(innerShadow);
    }

    function createMarbleGroup(px, py, value, selected) {
        const marbleR = HEX_SIZE * 0.50;
        const g = createSVG('g', { 'class': 'marble-group' });

        // Selection glow (behind marble)
        if (selected) {
            g.appendChild(createSVG('circle', {
                cx: px, cy: py, r: marbleR + 8,
                fill: 'none', stroke: COLOR_SELECTED, 'stroke-width': '6',
                opacity: '0.3', 'class': 'selection-glow',
            }));
            g.appendChild(createSVG('circle', {
                cx: px, cy: py, r: marbleR + 3,
                fill: 'none', stroke: COLOR_SELECTED, 'stroke-width': '2.5',
                'class': 'selection-ring',
            }));
        }

        // Shadow
        g.appendChild(createSVG('circle', {
            cx: px + 1, cy: py + 2, r: marbleR,
            fill: 'rgba(0,0,0,0.3)',
        }));

        // Marble
        const isBlack = value === 1;
        g.appendChild(createSVG('circle', {
            cx: px, cy: py, r: marbleR,
            fill: isBlack ? 'url(#grad-black)' : 'url(#grad-white)',
            stroke: isBlack ? '#111' : '#ccc',
            'stroke-width': '1',
            'class': 'marble ' + (isBlack ? 'black' : 'white'),
        }));

        return g;
    }

    function isMyInteractiveMarble(value) {
        return !gameOver && gameState && gameState.turn === myColor &&
            ((myColor === 'black' && value === 1) || (myColor === 'white' && value === 2));
    }

    function renderMarble(q, r, value) {
        const pos = hexToPixel(q, r);
        const selected = isSelected(q, r);
        const g = createMarbleGroup(pos.x, pos.y, value, selected);

        if (isMyInteractiveMarble(value)) {
            g.lastChild.classList.add('interactive');
            g.lastChild.setAttribute('data-q', q);
            g.lastChild.setAttribute('data-r', r);
            g.addEventListener('click', function(e) {
                e.stopPropagation();
                toggleSelection(q, r);
            });
        }

        boardDynamic.appendChild(g);
    }

    function renderMarbleAnimated(toQ, toR, value, fromQ, fromR) {
        const fromPos = hexToPixel(fromQ, fromR);
        const toPos = hexToPixel(toQ, toR);
        const selected = isSelected(toQ, toR);
        const g = createMarbleGroup(toPos.x, toPos.y, value, selected);

        if (isMyInteractiveMarble(value)) {
            g.lastChild.classList.add('interactive');
            g.lastChild.setAttribute('data-q', toQ);
            g.lastChild.setAttribute('data-r', toR);
            g.addEventListener('click', function(e) {
                e.stopPropagation();
                toggleSelection(toQ, toR);
            });
        }

        // Animate from old position
        const dx = fromPos.x - toPos.x;
        const dy = fromPos.y - toPos.y;
        g.style.transform = 'translate(' + dx + 'px, ' + dy + 'px)';
        g.style.transition = 'transform 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94)';

        boardDynamic.appendChild(g);

        requestAnimationFrame(function() {
            requestAnimationFrame(function() {
                g.style.transform = 'translate(0, 0)';
            });
        });
    }

    function renderMarblePushOff(fromQ, fromR, toPixelX, toPixelY, value) {
        const fromPos = hexToPixel(fromQ, fromR);
        const g = createMarbleGroup(fromPos.x, fromPos.y, value, false);
        g.classList.add('push-off');

        const dx = toPixelX - fromPos.x;
        const dy = toPixelY - fromPos.y;
        g.style.transform = 'translate(0, 0) scale(1)';
        g.style.opacity = '1';
        g.style.transition = 'transform 0.5s cubic-bezier(0.25, 0.46, 0.45, 0.94), opacity 0.5s ease-out';
        g.style.transformOrigin = fromPos.x + 'px ' + fromPos.y + 'px';

        boardDynamic.appendChild(g);

        requestAnimationFrame(function() {
            requestAnimationFrame(function() {
                g.style.transform = 'translate(' + dx + 'px, ' + dy + 'px) scale(0.3)';
                g.style.opacity = '0';
            });
        });
    }

    function renderDirectionArrows() {
        // Hide arrows during multi-select or when nothing selected
        if (shiftHeld || selectedMarbles.length === 0) return;

        // Compute centroid of selected marbles in hex space
        let cq = 0, cr = 0;
        selectedMarbles.forEach(function(m) {
            cq += m.q;
            cr += m.r;
        });
        cq /= selectedMarbles.length;
        cr /= selectedMarbles.length;

        DIRECTIONS.forEach(function (dir) {
            // Place arrow on the adjacent cell one hex step from the centroid
            const pos = hexToPixel(cq + dir.dq, cr + dir.dr);
            const ax = pos.x;
            const ay = pos.y;

            const arrowG = createSVG('g', {
                'class': 'direction-arrow enabled',
                'data-dq': dir.dq,
                'data-dr': dir.dr,
            });

            // Arrow background circle
            const bgCircle = createSVG('circle', {
                cx: ax,
                cy: ay,
                r: 13,
                fill: COLOR_ARROW,
                'fill-opacity': '0.85',
                stroke: '#fff',
                'stroke-width': '1.5',
                'stroke-opacity': '0.5',
                'class': 'arrow-bg',
            });
            arrowG.appendChild(bgCircle);

            // Arrow triangle pointing in direction
            const angleRad = dir.angle * Math.PI / 180;
            const triPoints = arrowTriangle(ax, ay, angleRad, 7);
            const tri = createSVG('polygon', {
                points: triPoints,
                fill: '#fff',
                'class': 'arrow-tri',
            });
            arrowG.appendChild(tri);

            arrowG.addEventListener('click', function (e) {
                e.stopPropagation();
                sendMove(dir.dq, dir.dr);
            });

            arrowG.addEventListener('mouseenter', function () {
                bgCircle.setAttribute('fill', COLOR_ARROW_HOVER);
                bgCircle.setAttribute('fill-opacity', '1');
            });
            arrowG.addEventListener('mouseleave', function () {
                bgCircle.setAttribute('fill', COLOR_ARROW);
                bgCircle.setAttribute('fill-opacity', '0.85');
            });

            boardDynamic.appendChild(arrowG);
        });
    }

    function arrowTriangle(cx, cy, angleRad, size) {
        // Triangle pointing in the given angle direction
        const pts = [];
        // Tip
        pts.push(
            (cx + size * Math.cos(angleRad)).toFixed(1) + ',' +
            (cy + size * Math.sin(angleRad)).toFixed(1)
        );
        // Left base
        pts.push(
            (cx + size * 0.7 * Math.cos(angleRad + 2.4)).toFixed(1) + ',' +
            (cy + size * 0.7 * Math.sin(angleRad + 2.4)).toFixed(1)
        );
        // Right base
        pts.push(
            (cx + size * 0.7 * Math.cos(angleRad - 2.4)).toFixed(1) + ',' +
            (cy + size * 0.7 * Math.sin(angleRad - 2.4)).toFixed(1)
        );
        return pts.join(' ');
    }

    // ---------------------
    // SVG helpers
    // ---------------------

    function createSVG(tag, attrs) {
        const el = document.createElementNS(SVG_NS, tag);
        if (attrs) {
            for (const key in attrs) {
                el.setAttribute(key, attrs[key]);
            }
        }
        return el;
    }

})();
