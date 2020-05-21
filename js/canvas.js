var canvas = {

    _moveState_none: 0,
    _moveState_swap: 1,
    _moveState_rect: 2,
    _moveState_drag: 3,
    _selection: {
        moveState: 0, // this._moveState_none,
        tileIds: {},
        isSeen: false, // seen or unseen tiles
        startX: 0,
        startY: 0,
        endX: 0,
        endY: 0,
    },
    _draw: {
        width: 0,
        height: 0,
        tileLength: 20,
        textOffset: 20 * 0.15, // tileLength * #
        unusedMinX: 0,
        unusedMinY: 0,
        usedMinX: 0,
        usedMinY: 0,
        numRows: 0,
        numCols: 0,
    },

    redraw: function () {
        this._draw.ctx.strokeStyle = "black";
        this._draw.ctx.fillStyle = "black";
        this._draw.ctx.clearRect(0, 0, this._draw.width, this._draw.height);
        this._draw.ctx.fillText("Unused Tiles:", 0, this._draw.unusedMinY - this._draw.textOffset);
        this._drawUnusedTiles(false);
        this._draw.ctx.fillText("Game Area:", 0, this._draw.usedMinY - this._draw.textOffset);
        this._draw.ctx.strokeRect(this._draw.usedMinX, this._draw.usedMinY,
            this._draw.numCols * this._draw.tileLength, this._draw.numRows * this._draw.tileLength)
        this._drawUsedTiles(false);
        if (this._selection.moveState == this._moveState_rect) {
            this._drawSelectionRectangle();
        } else if (Object.keys(this._selection.tileIds).length != 0) {
            this._draw.ctx.strokeStyle = "blue";
            this._draw.ctx.fillStyle = "blue";
            this._drawUnusedTiles(true);
            this._drawUsedTiles(true);
        }
    },

    _drawUnusedTiles: function (fromSelection) {
        for (var i = 0; i < game.unusedTileIds.length; i++) {
            var unusedTileId = game.unusedTileIds[i];
            var x = this._draw.unusedMinX + i * this._draw.tileLength;
            var y = this._draw.unusedMinY;
            var tile = game.unusedTiles[unusedTileId];
            this._drawTile(x, y, tile, fromSelection);
        }
    },

    _drawUsedTiles: function (fromSelection) {
        for (var c in game.usedTileLocs) {
            for (var r in game.usedTileLocs[c]) {
                var x = this._draw.usedMinX + c * this._draw.tileLength;
                var y = this._draw.usedMinY + r * this._draw.tileLength;
                var tile = game.usedTileLocs[c][r];
                this._drawTile(x, y, tile, fromSelection);
            }
        }
    },

    _getTileSelection: function (x, y) { // return { tile:{ id:int, ch:string }, isUsed:bool, x:int, y:int }
        // unused tile check
        if (this._draw.unusedMinX <= x && x < this._draw.unusedMinX + game.unusedTileIds.length * this._draw.tileLength
            && this._draw.unusedMinY <= y && y < this._draw.unusedMinY + this._draw.tileLength) {
            var idx = Math.floor((x - this._draw.unusedMinX)  / this._draw.tileLength);
            var id = game.unusedTileIds[idx];
            var tile = game.unusedTiles[id];
            if (tile != null) {
                return { tile: tile, isUsed: false };
            }
        }
        // used tile check
        else if (this._draw.usedMinX <= x && x < this._draw.usedMinX + this._draw.numCols * this._draw.tileLength
            && this._draw.usedMinY <= y && y < this._draw.usedMinY + this._draw.numRows * this._draw.tileLength) {
            var c = Math.floor((x - this._draw.usedMinX) / this._draw.tileLength);
            var r = Math.floor((y - this._draw.usedMinY) / this._draw.tileLength);
            if (game.usedTileLocs[c] != null && game.usedTileLocs[c][r] != null) {
                return { tile: game.usedTileLocs[c][r], isUsed: true, x: c, y: r };
            }
        }
        return null;
    },

    _drawSelectionRectangle: function () {
        var x = this._selection.startX < this._selection.endX ? this._selection.startX : this._selection.endX;
        var y = this._selection.startY < this._selection.endY ? this._selection.startY : this._selection.endY;
        var width = Math.abs(this._selection.endX - this._selection.startX);
        var height = Math.abs(this._selection.endY - this._selection.startY);
        this._draw.ctx.strokeRect(x, y, width, height);
    },

    _drawTile: function (x, y, tile, fromSelection) {
        if (fromSelection) {
            if (this._selection.tileIds[tile.id] == null) {
                return;
            }
            x += this._selection.endX - this._selection.startX;
            y += this._selection.endY - this._selection.startY;
        } else if (this._selection.moveState == this._moveState_drag
            && this._selection.tileIds[tile.id] != null) {
            return;
        }
        this._draw.ctx.strokeRect(x, y, this._draw.tileLength, this._draw.tileLength);
        this._draw.ctx.fillText(tile.ch,
            x + this._draw.textOffset,
            y + this._draw.tileLength - this._draw.textOffset);
    },

    _inSelectionRect: function (x, y) {
        var minX, minY, maxX, maxY
        [minX, maxX] = this._selection.startX < this._selection.endX
            ? [this._selection.startX, this._selection.endX]
            : [this._selection.endX, this._selection.startX];
        [minY, maxY] = this._selection.startY < this._selection.endY
            ? [this._selection.startY, this._selection.endY]
            : [this._selection.endY, this._selection.startY];
        return minX <= x && x < maxX
            && minY <= y && y < maxY;
    },

    _onMouseDown: function (event) {
        canvas._mouseDown(event.offsetX, event.offsetY);
    },

    _mouseDown: function (offsetX, offsetY) {
        this._selection.startX = this._selection.endX = offsetX;
        this._selection.startY = this._selection.endY = offsetY;
        var selectedTile = canvas._getTileSelection(offsetX, offsetY);
        var tileId;
        if (this._selection.moveState == this._moveState_swap) {
            if (selectedTile == null) {
                this._selection.moveState = this._moveState_none;
                log.info("swap cancelled");
                return;
            }
            tileId = selectedTile.tile.id;
            this._selection.tileIds[tileId] = true;
            return;
        }
        var hasPreviousSelection = Object.keys(this._selection.tileIds).length > 0;
        if (hasPreviousSelection) {
            if (selectedTile != null) {
                if (this._selection.tileIds[selectedTile.tile.id] == null) {
                    this._selection.tileIds = {};
                    this._selection.tileIds[selectedTile.tile.id] = true;
                }
                this._selection.moveState = this._moveState_drag;
            } else {
                this._selection.tileIds = {};
                this._selection._moveState = this._moveState_none;
                this.redraw();
            }
        } else if (selectedTile != null) {
            tileId = selectedTile.tile.id;
            this._selection.tileIds[tileId] = true;
            this._selection.moveState = this._moveState_drag;
        } else {
            this._selection.moveState = this._moveState_rect;
        }
    },

    _onMouseUp: function (event) {
        canvas._mouseUp(event.offsetX, event.offsetY);
    },

    _mouseUp: function (offsetX, offsetY) {
        if (this._selection.moveState == this._moveState_none) {
            return;
        }
        this._selection.endX = offsetX;
        this._selection.endY = offsetY;
        switch (this._selection.moveState) {
            case this._moveState_swap:
                this._selection.moveState = this._moveState_none;
                this._swap();
                break;
            case this._moveState_rect:
                this._selection.tileIds = this._getSelectedTileIds();
                this._selection.moveState = this._moveState_none;
                this._selection.startX = this._selection.endX = 0;
                this._selection.startY = this._selection.endY = 0;
                this.redraw();
                break;
            case this._moveState_drag:
                this._moveSelectedTiles();
                this._selection.tileIds = {};
                this._selection.moveState = this._moveState_none;
                this.redraw();
                break
        }
    },

    _onMouseMove: function (event) {
        canvas._mouseMove(event.offsetX, event.offsetY);
    },

    _mouseMove: function (offsetX, offsetY) {
        switch (this._selection.moveState) {
            case this._moveState_drag:
            case this._moveState_rect:
                this._selection.endX = offsetX;
                this._selection.endY = offsetY;
                this.redraw();
                break;
        }
    },

    _getSelectedTileIds: function () {
        this._selection.tileIds = {};
        var minX = this._selection.startX < this._selection.endX ? this._selection.startX : this._selection.endX;
        var maxX = this._selection.startX > this._selection.endX ? this._selection.startX : this._selection.endX;
        var minY = this._selection.startY < this._selection.endY ? this._selection.startY : this._selection.endY;
        var maxY = this._selection.startY > this._selection.endY ? this._selection.startY : this._selection.endY;
        var selectedUnusedTileIds = {};
        if (this._draw.unusedMinX <= maxX && minX < this._draw.unusedMinX + game.unusedTileIds.length * this._draw.tileLength
            && this._draw.unusedMinY <= maxY && minY < this._draw.unusedMinY + this._draw.tileLength) {
            selectedUnusedTileIds = this._getSelectedUnusedTileIds(minX, maxX, minY, maxY);
        }
        var selectedUsedTileIds = {}
        if (this._draw.usedMinX <= maxX && minX < this._draw.usedMinX + this._draw.numCols * this._draw.tileLength
            && this._draw.usedMinY <= maxY && minY < this._draw.usedMinY + this._draw.numRows * this._draw.tileLength) {
            selectedUsedTileIds = this._getSelectedUsedTileIds(minX, maxX, minY, maxY);
        }
        if (Object.keys(selectedUnusedTileIds).length != 0) {
            if (Object.keys(selectedUsedTileIds).length != 0) {
                return {}; // cannot select used and unused tiles
            }
            return selectedUnusedTileIds;
        }
        return selectedUsedTileIds;
    },

    _getSelectedUnusedTileIds: function (minX, maxX) {
        var minIndex = Math.floor((minX - this._draw.unusedMinX) / this._draw.tileLength);
        var maxIndex = Math.floor((maxX - this._draw.unusedMinX) / this._draw.tileLength);
        minIndex = Math.max(minIndex, 0);
        maxIndex = Math.min(maxIndex, game.unusedTileIds.length);
        var tileIds = {};
        for (var i = minIndex; i < maxIndex; i++) {
            var tileId = game.unusedTileIds[i];
            tileIds[tileId] = true;
        }
        return tileIds;
    },

    _getSelectedUsedTileIds: function (minX, maxX, minY, maxY) {
        var tileIds = {};
        var usedTilesX = Object.keys(game.usedTileLocs);
        for (var i = 0; i < usedTilesX.length; i++) {
            var c = parseInt(usedTilesX[i]);
            var usedTileLocsY = game.usedTileLocs[c];
            var usedTilesY = Object.keys(usedTileLocsY);
            for (var j = 0; j < usedTilesY.length; j++) {
                var r = parseInt(usedTilesY[j]);
                if (this._draw.usedMinX + c * this._draw.tileLength <= maxX
                    && minX < this._draw.usedMinX + (c + 1) * this._draw.tileLength
                    && this._draw.usedMinY + r * this._draw.tileLength <= maxY
                    && minY < this._draw.usedMinY + (r + 1) * this._draw.tileLength) {
                    var tileId = usedTileLocsY[r].id;
                    tileIds[tileId] = true;
                }
            }
        }
        return tileIds;
    },

    _moveSelectedTiles: function () {
        var tilePositions = this._getSelectionTilePositions();
        var i, tp;
        // if any of the new tile positions are currently used by non-moving tiles, do not change any
        for (i = 0; i < tilePositions.length; i++) {
            tp = tilePositions[i];
            if (game.usedTileLocs[tp.x] != null && game.usedTileLocs[tp.x][tp.y] != null) {
                var oldTile = game.usedTileLocs[tp.x][tp.y]
                if (this._selection.tileIds[oldTile.id] == null) {
                    tilePositions = [];
                    return;
                }
            }
        }
        for (i = 0; i < tilePositions.length; i++) {
            tp = tilePositions[i];
            // cleanup old position
            if (game.unusedTiles[tp.tile.id] != null) {
                delete game.unusedTiles[tp.tile.id];
                for (var j = 0; j < game.unusedTileIds.length; j++) {
                    if (game.unusedTileIds[j] == tp.tile.id) {
                        game.unusedTileIds.splice(j, 1);
                        break;
                    }
                }
            } else {
                var prevTp = game.usedTilePositions[tp.tile.id];
                if (prevTp.tile.id == game.usedTileLocs[prevTp.x][prevTp.y].id) {
                    delete game.usedTileLocs[prevTp.x][prevTp.y];
                    if (Object.keys(game.usedTileLocs[prevTp.x]).length == 0) {
                        delete game.usedTileLocs[prevTp.x];
                    }
                }
            }
            // update the tilePositions
            if (game.usedTileLocs[tp.x] == null) {
                game.usedTileLocs[tp.x] = {};
            }
            game.usedTileLocs[tp.x][tp.y] = tp.tile;
            game.usedTilePositions[tp.tile.id] = tp;
        }
        // send the message, redraw
        if (tilePositions.length > 0) {
            websocket.send({ type: 9, tilePositions: tilePositions }); // gameTilesMoved
        }
    },

    _getSelectionTilePositions: function () {
        var tileIds = Object.keys(this._selection.tileIds);
        if (tileIds.length == 0) {
            return [];
        }
        var endC = Math.floor((this._selection.endX - this._draw.usedMinX) / this._draw.tileLength);
        var endR = Math.floor((this._selection.endY - this._draw.usedMinY) / this._draw.tileLength);
        var centralTileSelection = this._getTileSelection(this._selection.startX, this._selection.startY);
        return centralTileSelection.isUsed
            ? this._getSelectionUsedTilePositions(tileIds, endC, endR, centralTileSelection.tile)
            : this._getSelectionUnusedTilePositions(tileIds, endC, endR, centralTileSelection.tile);
    },

    _getSelectionUnusedTilePositions: function (tileIds, endC, endR, centralTile) {
        if (endR < 0 || endR >= this._draw.numRows) {
            return [];
        }
        var tilePositions = [];
        var getUnusedTileIndex = function (tileId) {
            for (var i = 0; i < game.unusedTileIds.length; i++) {
                if (game.unusedTileIds[i] == tileId) {
                    return i;
                }
            }
            return -1;
        };
        var centralTileIdx = getUnusedTileIndex(centralTile.id);
        for (var i = 0; i < tileIds.length; i++) {
            var tileId = tileIds[i];
            var tileIdx = getUnusedTileIndex(tileId);
            var deltaCentralTileIdx = tileIdx - centralTileIdx;
            var tile = game.unusedTiles[tileId];
            var c = endC + deltaCentralTileIdx;
            if (c < 0 || c >= this._draw.numCols) {
                return [];
            }
            tilePositions.push({ tile: tile, x: c, y: endR });
        }
        return tilePositions;
    },

    _getSelectionUsedTilePositions: function (tileIds, endC, endR, centralTile) {
        var tilePositions = [];
        var centralTilePosition = game.usedTilePositions[centralTile.id]
        var deltaC = endC - centralTilePosition.x;
        var deltaR = endR - centralTilePosition.y;
        for (var i = 0; i < tileIds.length; i++) {
            var tileId = tileIds[i];
            var tp = game.usedTilePositions[tileId];
            var c = tp.x + deltaC;
            var r = tp.y + deltaR;
            if (c < 0 || c >= this._draw.numCols || r < 0 || r >= this._draw.numRows) {
                return [];
            }
            tilePositions.push({ tile: tp.tile, x: c, y: r, });
        }
        return tilePositions;
    },

    startSwap: function () {
        this._selection.moveState = this._moveState_swap;
        this._selection.tileIds = {};
        this.redraw();
    },

    _swap: function () {
        var selectedTile = canvas._getTileSelection(this._selection.endX, this._selection.endY);
        if (selectedTile == null || selectedTile.tile.id != Object.keys(this._selection.tileIds)[0]) {
            log.info("swap cancelled");
            return;
        }
        if (selectedTile.isUsed) {
            delete game.usedTileLocs[selectedTile.x][selectedTile.y];
            if (Object.keys(game.usedTileLocs[selectedTile.x]).length == 0) {
                delete game.usedTileLocs[selectedTile.x];
            }
            delete game.usedTilePositions[selectedTile.tile.id];
        } else {
            delete game.unusedTiles[selectedTile.tile.id];
            for (var i = 0; i < game.unusedTileIds.length; i++) {
                if (game.unusedTileIds[i] == selectedTile.tile.id) {
                    game.unusedTileIds.splice(i, 1);
                    break;
                }
            }
        }
        websocket.send({ type: 8, tiles: [selectedTile.tile] }); // gameSwap
    },

    init: function () {
        var canvasElement = document.getElementById("game-canvas");
        this._draw.ctx = canvasElement.getContext("2d");
        this._draw.width = canvasElement.width;
        this._draw.height = canvasElement.height;
        this._draw.ctx.font = this._draw.tileLength + 'px serif';
        this._draw.ctx.lineWidth = 1;
        var padding = 5;
        this._draw.unusedMinX = padding;
        this._draw.unusedMinY = this._draw.tileLength
        this._draw.usedMinX = padding;
        this._draw.usedMinY = this._draw.tileLength * 4;
        var usedMaxX = this._draw.width - padding;
        var usedMaxY = this._draw.height - padding;
        this._draw.numRows = Math.floor((usedMaxY - this._draw.usedMinY) / this._draw.tileLength);
        this._draw.numCols = Math.floor((usedMaxX - this._draw.usedMinX) / this._draw.tileLength);
        canvasElement.addEventListener("mousedown", this._onMouseDown);
        canvasElement.addEventListener("mouseup", this._onMouseUp);
        canvasElement.addEventListener("mousemove", this._onMouseMove)
    },
};