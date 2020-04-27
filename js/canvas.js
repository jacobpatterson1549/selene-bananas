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
        var unusedTileIds = Object.keys(game.unusedTiles);
        for (var i = 0; i < unusedTileIds.length; i++) {
            var unusedTileId = unusedTileIds[i];
            this._drawTile(ctx, i * tileLength, tileLength, game.unusedTiles[unusedTileId], tileLength, textOffset);
        }

        // draw used grid
        ctx.fillText("Game Area:", 0, tileLength * 4 - textOffset);
        var usedPadding = 5;
        var tileLengthRound = x => Math.ceil(x / tileLength) * tileLength
        var usedMinX = tileLengthRound(usedPadding);
        var usedMinY = tileLengthRound(usedPadding + tileLength * 4);
        var usedMaxX = tileLengthRound(width - usedPadding);
        var usedMaxY = tileLengthRound(height - usedPadding);
        var numRows = Math.floor((usedMaxY - usedMinY) / tileLength);
        var numCols = Math.floor((usedMaxX - usedMinX) / tileLength);
        // grid rows
        for (var i = 0; i <= numRows; i++) {
            ctx.moveTo(usedMinX, usedMinY + i * tileLength);
            ctx.lineTo(usedMaxX, usedMinY + i * tileLength);
        }
        // grid cols
        for (var i = 0; i <= numCols; i++) {
            ctx.moveTo(usedMinX + i * tileLength, usedMinY);
            ctx.lineTo(usedMinX + i * tileLength, usedMaxY);
        }
        // draw used tiles
        for (var c = 0; c < numCols; c++) { // x
            for (var r = 0; r < numRows; r++) { // y
                if (game.usedTileLocs[c] != null && game.usedTileLocs[c][r] != null) {
                    // TODO: this redraws the border, but it is not a big deal now.  Maybe later, the grid should not be drawn (only the border).
                    this._drawTile(ctx, usedMinX + c * tileLength, usedMinY + r * tileLength, game.usedTileLocs[c][r], tileLength, textOffset);
                }
            }
        }

        ctx.stroke();
        console.log("done drawing");
    },

    _getTile: function (x, y) { // return { tile:{ id:int, ch:string }, isUsed:bool, x:int, y:int }
        var canvasElement = document.getElementById("game-canvas");
        var width = canvasElement.width;
        var height = canvasElement.height;
        var tileLength = 20;
        // unused tile check
        var unusedTileIds = Object.keys(game.unusedTiles)
        if (x >= 0 && x < unusedTileIds.length * tileLength && y >= tileLength && y <= 2 * tileLength) {
            var idx = Math.floor(x / tileLength);
            var id = unusedTileIds[idx];
            var tile = game.unusedTiles[id];
            console.log("selected unused tile: ", tile.ch);
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
            var c  = Math.floor((x - usedMinX) / tileLength);
            var r  = Math.floor((y - usedMinY) / tileLength);
            var tile = game.usedTileLocs[c] ? game.usedTileLocs[c][r] : null;
            console.log("selected unused tile: ", (tile ? tile.ch : '-'), ", row ", r, ", col ", c);
            return { tile: tile, isUsed: true, x: c, y: r};
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
        if (selectedTile.tile == null) {
            if (canvas.isSwap) {
                this.log("error", "invalid swap");
                canvas.isSwap = false;
            }
            return;
        }
        var destinationTile = canvas._getTile(event.offsetX, event.offsetY);
        if (canvas.isSwap) {
            canvas.isSwap = false;
            canvas._swap(_selectedTile, destinationTile);
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
            tilePositions.push({tile: selectedTile.tile, x: selectedTile.x, y: selectedTile.y});
        }
        tilePositions.push({tile: selectedTile.tile, x: destinationTile.x, y: destinationTile.y});
        websocket.send({ type: 9, tilePositions: tilePositions }); // gameTileMoved
    },

    // TODO: onMouseMove

    _swap: function (src, dest) {
        if (src == null || dest == null || src.tile == null || dest.tile == null || src.tile.id != dest.tile.id) {
            this.log("error", "invalid swap");
            return;
        }
        if (dest.isUsed) {
            delete game.usedTileLocs[dest.x][dest.y];
            delete game.usedTiles[dest.tile.id];
        } else {
            delete game.unusedTiles[dest.tile.id];
        }
        websocket.send({ type: 8, tiles: [dest.tile] }); // gameSwap
    },

    init: function () {
        var canvasElement = document.getElementById("game-canvas");
        canvasElement.addEventListener("mousedown", this._onMouseDown);
        canvasElement.addEventListener("mouseup", this._onMouseUp);
    }
};

canvas.init();