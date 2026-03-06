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
    const COLOR_BLACK_MARBLE = '#2a2a2a';
    const COLOR_BLACK_MARBLE_SHINE = '#555555';
    const COLOR_WHITE_MARBLE = '#f0f0f0';
    const COLOR_WHITE_MARBLE_SHINE = '#ffffff';
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
    // (animation state is computed locally inside renderBoard)

    // SVG dimensions
    const SVG_WIDTH = 520;
    const SVG_HEIGHT = 480;
    const CENTER_X = SVG_WIDTH / 2;
    const CENTER_Y = SVG_HEIGHT / 2;

    // ---------------------
    // DOM references (set on init)
    // ---------------------

    let screenLobby, screenWaiting, screenGame, screenRules;
    let codeDisplay, codeValue;
    let btnHost, btnJoin, inputCode;
    let svgBoard;
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

        codeDisplay = document.getElementById('code-display');
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
                    var btn = document.getElementById('btn-copy-code');
                    btn.textContent = 'Copied!';
                    setTimeout(function () { btn.textContent = 'Copy'; }, 1500);
                });
            }
        });

        showScreen('lobby');
    }

    // ---------------------
    // Screen management
    // ---------------------

    function showScreen(name) {
        var screens = [screenLobby, screenWaiting, screenGame];
        var target = null;

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
        var code = inputCode.value.trim().toLowerCase();
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

        var protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        var url = protocol + '//' + location.host + '/ws?action=' + action;
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

            case 'state':
                var oldBoard = gameState ? Object.assign({}, gameState.board) : null;
                gameState = msg;
                selectedMarbles = [];
                renderBoard(oldBoard);
                updateStatus();
                var statusBar = document.querySelector('.status-bar');
                statusBar.classList.remove('turn-flash');
                void statusBar.offsetWidth; // force reflow
                statusBar.classList.add('turn-flash');
                break;

            case 'game_over':
                var oldBoard = gameState ? Object.assign({}, gameState.board) : null;
                gameState = msg;
                gameOver = true;
                selectedMarbles = [];
                renderBoard(oldBoard);
                updateStatus();
                showGameOver(msg.winner);
                break;

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

        var colorIcon = myColor === 'black' ? '⚫' : '⚪';
        statusColor.textContent = 'You are ' + colorIcon + ' ' + capitalize(myColor);

        var isMyTurn = gameState.turn === myColor;
        statusTurn.textContent = isMyTurn ? 'Your turn' : 'Opponent\'s turn';
        statusTurn.className = 'status-turn' + (isMyTurn ? ' my-turn' : '');

        var capB = gameState.captured ? gameState.captured.black : 0;
        var capW = gameState.captured ? gameState.captured.white : 0;

        statusCaptured.innerHTML = buildCapturedTray(capB, capW);
        statusMessage.textContent = '';
    }

    function buildCapturedTray(capBlack, capWhite) {
        var html = '<span class="cap-tray">';
        // Black captured (by white)
        for (var i = 0; i < 6; i++) {
            var filled = i < capBlack;
            html += '<span class="cap-marble cap-marble-black' + (filled ? ' filled' : '') + '"></span>';
        }
        html += '<span class="cap-sep">|</span>';
        // White captured (by black)
        for (var i = 0; i < 6; i++) {
            var filled = i < capWhite;
            html += '<span class="cap-marble cap-marble-white' + (filled ? ' filled' : '') + '"></span>';
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
        var isWinner = winner === myColor;
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
        var x = HEX_SIZE * (3 / 2 * q);
        var y = HEX_SIZE * (Math.sqrt(3) / 2 * q + Math.sqrt(3) * r);
        return { x: x + CENTER_X, y: y + CENTER_Y };
    }

    // Generate the 6 vertices of a flat-top hexagon
    function hexCorners(cx, cy, size) {
        var corners = [];
        for (var i = 0; i < 6; i++) {
            var angle = Math.PI / 180 * (60 * i);
            corners.push({
                x: cx + size * Math.cos(angle),
                y: cy + size * Math.sin(angle)
            });
        }
        return corners;
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
        // Can't select if not our turn or game over
        if (!gameState || gameState.turn !== myColor || gameOver) return;

        // Can only select our own marbles
        var val = getCellValue(q, r);
        var myVal = myColor === 'black' ? 1 : 2;
        if (val !== myVal) {
            selectedMarbles = [];
            renderBoard();
            return;
        }

        // If already selected, deselect
        if (isSelected(q, r)) {
            selectedMarbles = selectedMarbles.filter(function (m) {
                return m.q !== q || m.r !== r;
            });
            renderBoard();
            return;
        }

        // Try adding to selection
        var next = selectedMarbles.concat([{ q: q, r: r }]);

        if (next.length === 1) {
            selectedMarbles = next;
        } else if (next.length === 2) {
            if (areAdjacent(next[0], next[1])) {
                selectedMarbles = next;
            } else {
                // Start new selection
                selectedMarbles = [{ q: q, r: r }];
            }
        } else if (next.length === 3) {
            if (areCollinear(next)) {
                selectedMarbles = sortMarbles(next);
            } else {
                // Start new selection
                selectedMarbles = [{ q: q, r: r }];
            }
        } else {
            // Already have 3, start new selection
            selectedMarbles = [{ q: q, r: r }];
        }

        renderBoard();
    }

    function areAdjacent(a, b) {
        var dq = b.q - a.q;
        var dr = b.r - a.r;
        return DIRECTIONS.some(function (d) {
            return d.dq === dq && d.dr === dr;
        });
    }

    function areCollinear(marbles) {
        if (marbles.length <= 1) return true;
        if (marbles.length === 2) return areAdjacent(marbles[0], marbles[1]);

        // For 3 marbles: sort them and check they form a line
        var sorted = sortMarbles(marbles);
        var dq1 = sorted[1].q - sorted[0].q;
        var dr1 = sorted[1].r - sorted[0].r;
        var dq2 = sorted[2].q - sorted[1].q;
        var dr2 = sorted[2].r - sorted[1].r;

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

        var msg = {
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
        // Clear SVG
        while (svgBoard.firstChild) {
            svgBoard.removeChild(svgBoard.firstChild);
        }

        // Create defs for gradients
        var defs = createSVG('defs');
        svgBoard.appendChild(defs);

        // Black marble gradient — 3 stops for depth
        var blackGrad = createSVG('radialGradient', {
            id: 'grad-black', cx: '50%', cy: '50%', r: '50%', fx: '35%', fy: '30%',
        });
        blackGrad.appendChild(createSVG('stop', { offset: '0%', 'stop-color': '#777' }));
        blackGrad.appendChild(createSVG('stop', { offset: '50%', 'stop-color': '#333' }));
        blackGrad.appendChild(createSVG('stop', { offset: '100%', 'stop-color': '#111' }));
        defs.appendChild(blackGrad);

        // White marble gradient — 3 stops for depth
        var whiteGrad = createSVG('radialGradient', {
            id: 'grad-white', cx: '50%', cy: '50%', r: '50%', fx: '35%', fy: '30%',
        });
        whiteGrad.appendChild(createSVG('stop', { offset: '0%', 'stop-color': '#fff' }));
        whiteGrad.appendChild(createSVG('stop', { offset: '40%', 'stop-color': '#eee' }));
        whiteGrad.appendChild(createSVG('stop', { offset: '100%', 'stop-color': '#bbb' }));
        defs.appendChild(whiteGrad);

        // Selected highlight gradient
        var selGrad = createRadialGradient('grad-selected', 'rgba(255,215,0,0.4)', 'rgba(255,215,0,0)', '0%', '0%');
        defs.appendChild(selGrad);

        // Board glow filter
        var glowFilter = createSVG('filter', { id: 'board-glow', x: '-20%', y: '-20%', width: '140%', height: '140%' });
        glowFilter.appendChild(createSVG('feGaussianBlur', { 'in': 'SourceGraphic', stdDeviation: '8' }));
        defs.appendChild(glowFilter);

        // Board background (large hex shape)
        renderBoardBackground();

        // Render cells
        for (var q = -BOARD_RADIUS; q <= BOARD_RADIUS; q++) {
            for (var r = -BOARD_RADIUS; r <= BOARD_RADIUS; r++) {
                if (!isValidHex(q, r)) continue;
                renderCell(q, r);
            }
        }

        // Compute move direction from board diff (for animation)
        var moveDir = null;
        var newPositions = (gameState && gameState.board) ? gameState.board : {};

        if (oldBoard) {
            moveDir = detectMoveDirection(oldBoard, newPositions);
        }

        // Render marbles with animation
        if (gameState && gameState.board) {
            for (var key in newPositions) {
                var parts = key.split(',');
                var mq = parseInt(parts[0]);
                var mr = parseInt(parts[1]);
                var val = newPositions[key];

                if (oldBoard) {
                    // If same marble was already at this position, no animation
                    if (oldBoard[key] === val) {
                        renderMarble(mq, mr, val);
                        continue;
                    }
                    // New position for this color — animate from source using direction
                    if (moveDir) {
                        renderMarbleAnimated(mq, mr, val, mq - moveDir.dq, mr - moveDir.dr);
                        continue;
                    }
                }

                renderMarble(mq, mr, val);
            }
        }

        // Render direction arrows if marbles are selected
        if (selectedMarbles.length > 0) {
            renderDirectionArrows();
        }
    }

    // Detect the uniform move direction by comparing old and new board states.
    // Returns {dq, dr} or null if no direction can be determined.
    function detectMoveDirection(oldBoard, newPositions) {
        // Collect new and vacated positions for each color
        for (var ci = 1; ci <= 2; ci++) {
            var newPos = [];
            var vacated = {};
            var vacatedCount = 0;

            for (var key in newPositions) {
                if (newPositions[key] === ci && oldBoard[key] !== ci) {
                    var parts = key.split(',');
                    newPos.push({ q: parseInt(parts[0]), r: parseInt(parts[1]) });
                }
            }
            for (var key in oldBoard) {
                if (oldBoard[key] === ci && newPositions[key] !== ci) {
                    vacated[key] = true;
                    vacatedCount++;
                }
            }

            if (newPos.length === 0) continue;

            // Try each of the 6 directions
            for (var di = 0; di < DIRECTIONS.length; di++) {
                var dir = DIRECTIONS[di];
                var match = true;
                for (var ni = 0; ni < newPos.length; ni++) {
                    var srcKey = (newPos[ni].q - dir.dq) + ',' + (newPos[ni].r - dir.dr);
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
        var bgSize = HEX_SIZE * (BOARD_RADIUS + 0.8) * Math.sqrt(3);
        var corners = [];
        for (var i = 0; i < 6; i++) {
            var angle = Math.PI / 180 * (60 * i + 30);
            corners.push(CENTER_X + bgSize * Math.cos(angle) + ',' + (CENTER_Y + bgSize * Math.sin(angle)));
        }

        // Glow layer behind the board
        var glowBg = createSVG('polygon', {
            points: corners.join(' '),
            fill: '#3a7a22',
            opacity: '0.15',
            filter: 'url(#board-glow)',
        });
        svgBoard.appendChild(glowBg);

        var bg = createSVG('polygon', {
            points: corners.join(' '),
            fill: COLOR_BOARD_BG,
            stroke: '#1a3a0a',
            'stroke-width': '3'
        });
        svgBoard.appendChild(bg);
    }

    function renderCell(q, r) {
        var pos = hexToPixel(q, r);
        var cellSize = HEX_SIZE * 0.88;
        var cellR = cellSize * 0.62;

        // Outer rim (lighter edge for 3D pit effect)
        var rim = createSVG('circle', {
            cx: pos.x, cy: pos.y - 1, r: cellR,
            fill: '#4a8a2e',
            opacity: '0.5',
        });
        svgBoard.appendChild(rim);

        // Main pit
        var cell = createSVG('circle', {
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

        svgBoard.appendChild(cell);

        // Inner shadow (darker center for depth)
        var innerShadow = createSVG('circle', {
            cx: pos.x, cy: pos.y + 1, r: cellR * 0.85,
            fill: '#2a5014',
            opacity: '0.4',
            'pointer-events': 'none',
        });
        svgBoard.appendChild(innerShadow);
    }

    function renderMarble(q, r, value) {
        var pos = hexToPixel(q, r);
        var marbleR = HEX_SIZE * 0.50;
        var selected = isSelected(q, r);
        var isInteractive = !gameOver && gameState && gameState.turn === myColor &&
            ((myColor === 'black' && value === 1) || (myColor === 'white' && value === 2));

        // Marble group
        var g = createSVG('g', { 'class': 'marble-group', 'data-q': q, 'data-r': r });

        // Selection glow (drawn first, behind marble)
        if (selected) {
            // Outer glow
            var glow = createSVG('circle', {
                cx: pos.x, cy: pos.y, r: marbleR + 8,
                fill: 'none',
                stroke: COLOR_SELECTED,
                'stroke-width': '6',
                opacity: '0.3',
                'class': 'selection-glow',
            });
            g.appendChild(glow);

            // Inner ring
            var ring = createSVG('circle', {
                cx: pos.x, cy: pos.y, r: marbleR + 3,
                fill: 'none',
                stroke: COLOR_SELECTED,
                'stroke-width': '2.5',
                'class': 'selection-ring',
            });
            g.appendChild(ring);
        }

        // Shadow
        var shadow = createSVG('circle', {
            cx: pos.x + 1,
            cy: pos.y + 2,
            r: marbleR,
            fill: 'rgba(0,0,0,0.3)',
        });
        g.appendChild(shadow);

        // Marble
        var gradId = value === 1 ? 'url(#grad-black)' : 'url(#grad-white)';
        var marble = createSVG('circle', {
            cx: pos.x,
            cy: pos.y,
            r: marbleR,
            fill: gradId,
            stroke: value === 1 ? '#111' : '#ccc',
            'stroke-width': '1',
            'class': 'marble ' + (value === 1 ? 'black' : 'white') + (isInteractive ? ' interactive' : ''),
            'data-q': q,
            'data-r': r,
        });

        marble.addEventListener('click', function (e) {
            e.stopPropagation();
            toggleSelection(q, r);
        });

        g.appendChild(marble);
        svgBoard.appendChild(g);
    }

    function renderMarbleAnimated(toQ, toR, value, fromQ, fromR) {
        var fromPos = hexToPixel(fromQ, fromR);
        var toPos = hexToPixel(toQ, toR);
        var marbleR = HEX_SIZE * 0.50;
        var selected = isSelected(toQ, toR);
        var isInteractive = !gameOver && gameState && gameState.turn === myColor &&
            ((myColor === 'black' && value === 1) || (myColor === 'white' && value === 2));

        var g = createSVG('g', { 'class': 'marble-group' });

        // Calculate translation
        var dx = fromPos.x - toPos.x;
        var dy = fromPos.y - toPos.y;

        // Selection glow
        if (selected) {
            var glow = createSVG('circle', {
                cx: toPos.x, cy: toPos.y, r: marbleR + 8,
                fill: 'none', stroke: COLOR_SELECTED, 'stroke-width': '6',
                opacity: '0.3', 'class': 'selection-glow',
            });
            g.appendChild(glow);

            var ring = createSVG('circle', {
                cx: toPos.x, cy: toPos.y, r: marbleR + 3,
                fill: 'none', stroke: COLOR_SELECTED, 'stroke-width': '2.5',
                'class': 'selection-ring',
            });
            g.appendChild(ring);
        }

        // Shadow
        var shadow = createSVG('circle', {
            cx: toPos.x + 1, cy: toPos.y + 2, r: marbleR,
            fill: 'rgba(0,0,0,0.3)',
        });
        g.appendChild(shadow);

        // Marble
        var gradId = value === 1 ? 'url(#grad-black)' : 'url(#grad-white)';
        var marble = createSVG('circle', {
            cx: toPos.x, cy: toPos.y, r: marbleR,
            fill: gradId,
            stroke: value === 1 ? '#111' : '#ccc',
            'stroke-width': '1',
            'class': 'marble ' + (value === 1 ? 'black' : 'white') + (isInteractive ? ' interactive' : ''),
            'data-q': toQ, 'data-r': toR,
        });

        marble.addEventListener('click', function (e) {
            e.stopPropagation();
            toggleSelection(toQ, toR);
        });
        g.appendChild(marble);

        // Apply CSS animation via transform
        g.style.transform = 'translate(' + dx + 'px, ' + dy + 'px)';
        g.style.transition = 'transform 0.3s cubic-bezier(0.25, 0.46, 0.45, 0.94)';

        svgBoard.appendChild(g);

        // Trigger animation on next frame
        requestAnimationFrame(function() {
            requestAnimationFrame(function() {
                g.style.transform = 'translate(0, 0)';
            });
        });
    }

    function renderDirectionArrows() {
        // Compute centroid of selected marbles
        var cx = 0, cy = 0;
        selectedMarbles.forEach(function (m) {
            var pos = hexToPixel(m.q, m.r);
            cx += pos.x;
            cy += pos.y;
        });
        cx /= selectedMarbles.length;
        cy /= selectedMarbles.length;

        var arrowDist = HEX_SIZE * 2.3;

        DIRECTIONS.forEach(function (dir) {
            // Arrow position: offset from centroid in the hex direction
            var angleRad = dir.angle * Math.PI / 180;
            var ax = cx + arrowDist * Math.cos(angleRad);
            var ay = cy + arrowDist * Math.sin(angleRad);

            var arrowG = createSVG('g', {
                'class': 'direction-arrow',
                'data-dq': dir.dq,
                'data-dr': dir.dr,
            });

            // Arrow background circle
            var bgCircle = createSVG('circle', {
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
            var triPoints = arrowTriangle(ax, ay, angleRad, 7);
            var tri = createSVG('polygon', {
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

            svgBoard.appendChild(arrowG);
        });
    }

    function arrowTriangle(cx, cy, angleRad, size) {
        // Triangle pointing in the given angle direction
        var pts = [];
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
        var el = document.createElementNS(SVG_NS, tag);
        if (attrs) {
            for (var key in attrs) {
                el.setAttribute(key, attrs[key]);
            }
        }
        return el;
    }

    function createRadialGradient(id, colorCenter, colorEdge, fx, fy) {
        var grad = createSVG('radialGradient', {
            id: id,
            cx: '50%',
            cy: '50%',
            r: '50%',
            fx: fx,
            fy: fy,
        });
        var stop1 = createSVG('stop', {
            offset: '0%',
            'stop-color': colorCenter,
        });
        var stop2 = createSVG('stop', {
            offset: '100%',
            'stop-color': colorEdge,
        });
        grad.appendChild(stop1);
        grad.appendChild(stop2);
        return grad;
    }

})();
