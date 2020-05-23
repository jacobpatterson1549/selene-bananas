var game = {

    // tile { id, ch }
    // tilePosition { tile, x, y }
    unusedTiles: {}, // id => tile
    unusedTileIds: [], // id
    usedTilePositions: {}, // id => tilePosition
    usedTileLocs: {}, // x => y => tile

    _resetTiles: function () {
        this.unusedTiles = {};
        this.unusedTileIds = [];
        this.usedTileLocs = {};
    },

    create: function () {
        this._resetTiles();
        websocket.send({ type: 1 }); // gameCreate
    },

    join: function (event) {
        var joinGameButton = event.srcElement;
        var gameIdInput = joinGameButton.previousElementSibling;
        var gameId = parseInt(gameIdInput.value);
        this._resetTiles();
        websocket.send({ type: 2, gameID: gameId }); // gameJoin
    },

    leave: function () {
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = false;
        var lobbyTab = document.getElementById("tab-4");
        lobbyTab.checked = true;
    },

    delete: function () {
        var result = window.confirm("Are you sure? Deleting the game will kick everyone out.");
        if (!result) {
            return;
        }
        websocket.send({ type: 4 }); // gameDelete
    },

    start: function () {
        websocket.send({ type: 5, gameStatus: 1 }); // gameStatusChange, gameInProgress
    },

    finish: function () {
        websocket.send({ type: 5, gameStatus: 2 }); // gameStatusChange, gameFinished
    },

    snagTile: function () {
        websocket.send({ type: 7 }); // gameSnag
    },

    swapTile: function () {
        log.info("click a tile to swap for three others from the pile");
        canvas.startSwap();
    },

    _setTabActive: function () {
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = true;
        var gameTab = document.getElementById("tab-5");
        gameTab.checked = true;
    },

    replaceGameTiles: function (unusedTiles, usedTileLocs, silent) {
        this.unusedTiles = {};
        this.unusedTileIds = [];
        this.usedTilePositions = {};
        this.usedTileLocs = {};
        this.addUnusedTiles(unusedTiles, silent);
        if (usedTileLocs != null) {
            for (var i = 0; i < usedTileLocs.length; i++) {
                var tp = usedTileLocs[i]
                this.usedTilePositions[tp.tile.id] = tp;
                if (this.usedTileLocs[tp.x] == null) {
                    this.usedTileLocs[tp.x] = {};
                }
                this.usedTileLocs[tp.x][tp.y] = tp.tile;
            }
        }
        canvas.redraw();
        this._setTabActive();
    },

    addUnusedTiles: function (unusedTiles, silent) {
        var tileStrings = [];
        if (unusedTiles != null) {
            tileStrings.length = unusedTiles.length;
            for (var i = 0; i < unusedTiles.length; i++) {
                var t = unusedTiles[i];
                tileStrings[i] = " \"" + t.ch + "\"";
                this.unusedTiles[t.id] = t;
                this.unusedTileIds.push(t.id);
            }
        }
        if (silent == null || !silent) {
            log.info("adding unused tile" + (tileStrings.length == 1 ? "" : "s") + ": " + tileStrings);
        }
        canvas.redraw();
        this._setTabActive();
    },

    setStatus: function (status, tilesLeft) {
        var stateElement = document.querySelector("input#game-status");
        if (status) {
            switch (status) {
                case 3: // gameNotStarted
                    stateElement.value = "Not Started";
                    this._setButtonDisabled("game-snag", true);
                    this._setButtonDisabled("game-swap", true);
                    this._setButtonDisabled("game-start", false);
                    this._setButtonDisabled("game-finish", true);
                    break;
                case 1: // gameInProgress
                    stateElement.value = "In Progress";
                    this._setButtonDisabled("game-snag", false);
                    this._setButtonDisabled("game-swap", false);
                    this._setButtonDisabled("game-start", true);
                    this._setButtonDisabled("game-finish", true);
                    break;
                case 2: // gameFinished
                    stateElement.value = "Finished";
                    this._setButtonDisabled("game-snag", true);
                    this._setButtonDisabled("game-swap", true);
                    this._setButtonDisabled("game-start", true);
                    this._setButtonDisabled("game-finish", true);
                    break;
            }
            game._setInProgress(status == 1); // gameInProgress
        }
        if (tilesLeft != null) {
            this._setTilesLeft(tilesLeft);
        }
        if (status == 2) {
            this._setButtonDisabled("game-finish", true);
        }
    },

    _setTilesLeft: function (tilesLeft) {
        var tilesLeftElement = document.querySelector("input#game-tiles-left");
        tilesLeftElement.value = tilesLeft;
        if (tilesLeft == 0) {
            this._setButtonDisabled("game-snag", true);
            this._setButtonDisabled("game-swap", true);
            this._setButtonDisabled("game-finish", false);
        }
    },

    setPlayers: function (players) {
        var playersElement = document.querySelector("input#game-players");
        playersElement.value = players;
    },

    _setButtonDisabled(buttonElementId, state) {
        var buttonElement = document.querySelector("button#" + buttonElementId);
        buttonElement.disabled = state;
    },

    sendChat: function (event) {
        event.preventDefault();
        var gameChatElement = document.querySelector("input#game-chat");
        var message = gameChatElement.value;
        gameChatElement.value = "";
        websocket.send({ type: 18, info: message }); // game.Chat
    },

    isInProgress: function () {
        var inProgressElement = document.querySelector("#game>input.in-progress");
        return inProgressElement.checked;
    },

    _setInProgress: function (checked) {
        var inProgressElement = document.querySelector("#game>input.in-progress");
        inProgressElement.checked = checked;
    }
};