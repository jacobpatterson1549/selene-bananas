var game = {

    unusedTiles: {},
    unusedTileIds: [],
    usedTileLocs: {},

    _resetTiles: function () {
        this.unusedTiles = {};
        this.unusedTileIds = [];
        this.usedTileLocs = {};
    },

    create: function (event) {
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

    leave: function (event) {
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = false;
        var lobbyTab = document.getElementById("tab-4");
        lobbyTab.checked = true;
    },

    delete: function (event) {
        var result = window.confirm("Are you sure? Deleting the game will kick everyone out.");
        if (!result) {
            return;
        }
        websocket.send({ type: 4 }); // gameDelete
    },

    start: function (event) {
        websocket.send({ type: 5, gameState: 1 }); // gameStateChange, gameInProgress
    },

    finish: function (event) {
        websocket.send({ type: 5, gameState: 2 }); // gameStateChange, gameFinished
    },

    snagTile: function (event) {
        websocket.send({ type: 7 }); // gameSnag
    },

    swapTile: function (event) {
        log.info("click a tile to swap for three others from the pile");
        canvas.isSwap = true;
    },

    _setTabActive: function () {
        var hasGameElement = document.getElementById("has-game");
        hasGameElement.checked = true;
        var gameTab = document.getElementById("tab-5");
        gameTab.checked = true;
    },

    replaceGameTiles: function (unusedTiles, usedTileLocs, silent) {
        this.unusedTiles = {}
        this.usedTileLocs = {}
        this.addUnusedTiles(unusedTiles, true);
        if (usedTileLocs != null) {
            for (var i = 0; i < usedTileLocs.length; i++) {
                var tp = usedTileLocs[i]
                this.usedTiles[tp.id] = tp;
                if (this.usedTileLocs[tp.x] == null) {
                    this.usedTileLocs[tp.x] = {};
                }
                this.usedTileLocs[tp.x][tp.y] = tp.tile;
            }
        }
        if (silent == null || !silent) {
            canvas.redraw();
            this._setTabActive();
        }
    },

    addUnusedTiles: function (unusedTiles, silent) {
        var tileStrings = []
        if (unusedTiles != null) {
            tileStrings.length = unusedTiles.length;
            for (var i = 0; i < unusedTiles.length; i++) {
                var t = unusedTiles[i];
                tileStrings[i] = t.ch;
                this.unusedTiles[t.id] = t;
                this.unusedTileIds.push(t.id);
            }
        }
        if (silent == null || !silent) {
            log.info("adding " + tileStrings + " unused tiles");
            canvas.redraw();
            this._setTabActive();
        }
    },

    setState: function (state) {
        var stateElement = document.querySelector("input#game-state");
        switch (state) {
            case 3: // gameNotStarted
                stateElement.value = "Not Started"
                this._setButtonDisabled("game-snag", true);
                this._setButtonDisabled("game-swap", true);
                this._setButtonDisabled("game-start", false);
                this._setButtonDisabled("game-finish", true);
                break;
            case 1: // gameInProgress
                stateElement.value = "In Progress"
                this._setButtonDisabled("game-snag", false);
                this._setButtonDisabled("game-swap", false);
                this._setButtonDisabled("game-start", true);
                this._setButtonDisabled("game-finish", true);
                break;
            case 2: // gameFinished
                stateElement.value = "Finished"
                this._setButtonDisabled("game-snag", true);
                this._setButtonDisabled("game-swap", true);
                this._setButtonDisabled("game-start", true);
                this._setButtonDisabled("game-finish", true);
                break;
            default:
                log.error("invalid gameState: ", state);
                break;
        }
    },

    setTilesLeft: function (tilesLeft) {
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
        websocket.send({ type: 18, info: message }) // gameChatRecv
    },
};