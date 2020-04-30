var canvas = {

    // TODO: add flag to existing tiles when being moved tso only moved tile is drawn.
    _selectedTile: {}, // { tile:{ id:int, ch:string }, isUsed:bool, x:int, y:int }
    isSwap: false,

    redraw: function () {
        var canvasElement = document.getElementById("game-canvas");
        var ctx = canvasElement.getContext("2d");
        var width = canvasElement.width;
        var height = canvasElement.height;
        ctx.clearRect(0, 0, width, height);

        // TODO: use variables, share with _getTile(x,y)
        var tileLength = 20;
        var textOffset = tileLength * 0.15;
        ctx.font = tileLength + 'px serif';
        ctx.lineWidth = 1;

        // draw unused tiles
        ctx.fillText("Unused Tiles:", 0, tileLength - textOffset);
        for (var i = 0; i < game.unusedTileIds.length; i++) {
            var unusedTileId = game.unusedTileIds[i];
            var tile = game.unusedTiles[unusedTileId];
            this._drawTile(ctx, i * tileLength, tileLength, tile, tileLength, textOffset);
        }

        ctx.fillText("Game Area:", 0, tileLength * 4 - textOffset);
        var usedPadding = 5;
        var tileLengthRound = x => Math.ceil(x / tileLength) * tileLength
        var usedMinX = tileLengthRound(usedPadding);
        var usedMinY = tileLengthRound(usedPadding + tileLength * 4);
        var usedMaxX = tileLengthRound(width - usedPadding);
        var usedMaxY = tileLengthRound(height - usedPadding);
        ctx.moveTo(usedMinX, usedMinY);
        ctx.lineTo(usedMinX, usedMaxX);
        ctx.lineTo(usedMaxX, usedMaxY);
        ctx.lineTo(usedMaxX, usedMinY),
            ctx.lineTo(usedMinX, usedMinY);
        var numRows = Math.floor((usedMaxY - usedMinY) / tileLength);
        var numCols = Math.floor((usedMaxX - usedMinX) / tileLength);
        // draw used tiles
        for (var c = 0; c < numCols; c++) { // x
            for (var r = 0; r < numRows; r++) { // y
                if (game.usedTileLocs[c] != null && game.usedTileLocs[c][r] != null) {
                    this._drawTile(ctx, usedMinX + c * tileLength, usedMinY + r * tileLength, game.usedTileLocs[c][r], tileLength, textOffset);
                }
            }
        }

        ctx.stroke();
    },

    _getTile: function (x, y) { // return { tile:{ id:int, ch:string }, isUsed:bool, x:int, y:int }
        var canvasElement = document.getElementById("game-canvas");
        var width = canvasElement.width;
        var height = canvasElement.height;
        var tileLength = 20;
        // unused tile check
        if (x >= 0 && x < game.unusedTileIds.length * tileLength && y >= tileLength && y <= 2 * tileLength) {
            var idx = Math.floor(x / tileLength);
            var id = game.unusedTileIds[idx];
            var tile = game.unusedTiles[id];
            return { tile: tile, isUsed: false };
        }
        // used tile check
        var usedPadding = 5;
        var tileLengthRound = x => Math.ceil(x / tileLength) * tileLength
        var usedMinX = tileLengthRound(usedPadding);
        var usedMinY = tileLengthRound(usedPadding + tileLength * 4);
        var usedMaxX = tileLengthRound(width - usedPadding);
        var usedMaxY = tileLengthRound(height - usedPadding);
        if (x >= usedMinX && x <= usedMaxX && y >= usedMinY && y <= usedMaxY) {
            var c = Math.floor((x - usedMinX) / tileLength);
            var r = Math.floor((y - usedMinY) / tileLength);
            var tile = game.usedTileLocs[c] ? game.usedTileLocs[c][r] : null;
            return { tile: tile, isUsed: true, x: c, y: r };
        }

        return {};
    },

    _drawTile: function (ctx, x, y, tile, tileLength, textOffset) {
        ctx.strokeRect(x, y, tileLength, tileLength);
        ctx.fillText(tile.ch, x + textOffset, y + tileLength - textOffset);
    },

    _onMouseDown: function (event) {
        canvas._selectedTile = canvas._getTile(event.offsetX, event.offsetY);
    },

    _onMouseUp: function (event) {
        var selectedTile = canvas._selectedTile;
        canvas._selectedTile = null;
        var destinationTile = canvas._getTile(event.offsetX, event.offsetY);
        if (canvas.isSwap) {
            canvas.isSwap = false;
            canvas._swap(selectedTile, destinationTile);
            return;
        }
        if (selectedTile == null || selectedTile.tile == null) {
            if (canvas.isSwap) {
                canvas.isSwap = false;
            }
            return;
        }
        if (destinationTile.tile != null || !destinationTile.isUsed) {
            return;
        }
        if (game.usedTileLocs[destinationTile.x] != null && game.usedTileLocs[destinationTile.x][destinationTile.y] != null) {
            return;
        }
        // the tile drag is valid
        if (selectedTile.isUsed) {
            delete game.usedTileLocs[selectedTile.x][selectedTile.y];
        } else {
            delete game.unusedTiles[selectedTile.tile.id];
            for (var i = 0; i < game.unusedTileIds.length; i++) {
                if (game.unusedTileIds[i] == selectedTile.tile.id) {
                    game.unusedTileIds.splice(i, 1);
                    break;
                }
            }
        }
        game.usedTiles[selectedTile.tile.id] = selectedTile.tile;
        if (game.usedTileLocs[destinationTile.x] == null) {
            game.usedTileLocs[destinationTile.x] = {};
        }
        game.usedTileLocs[destinationTile.x][destinationTile.y] = selectedTile.tile;
        canvas.redraw();
        // send notification to server
        var tilePositions = [];
        if (selectedTile.isUsed) {
            tilePositions.push({ tile: selectedTile.tile, x: selectedTile.x, y: selectedTile.y });
        }
        tilePositions.push({ tile: selectedTile.tile, x: destinationTile.x, y: destinationTile.y });
        websocket.send({ type: 9, tilePositions: tilePositions }); // gameTileMoved
    },

    // TODO: onMouseMove

    _swap: function (src, dest) {
        if (src == null || dest == null) {
            return;
        }
        if (src.tile == null || dest.tile == null || src.tile.id != dest.tile.id) {
            log.info("swap cancelled");
            return;
        }
        if (src.isUsed) {
            delete game.usedTileLocs[src.x][src.y];
            delete game.usedTiles[src.tile.id];
        } else {
            delete game.unusedTiles[src.tile.id];
            for (var i = 0; i < game.unusedTileIds.length; i++) {
                if (game.unusedTileIds[i] == src.tile.id) {
                    game.unusedTileIds.splice(i, 1);
                    break;
                }
            }
        }
        websocket.send({ type: 8, tiles: [src.tile] }); // gameSwap
    },

    init: function () {
        var canvasElement = document.getElementById("game-canvas");
        canvasElement.addEventListener("mousedown", this._onMouseDown);
        canvasElement.addEventListener("mouseup", this._onMouseUp);
    }
};

canvas.init();