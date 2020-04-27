var canvas = {

    _tile_length: 10,

    redraw: function () {
        var unusedTiles = game.unusedTiles; // map[id]tile
        var usedTiles = game.usedTiles; // map[id]tile
        var usedTileLocs = game.usedTileLocs; //map[x][y]tilePosition

        var canvasElement = document.getElementById("game-canvas");
        var ctx = canvasElement.getContext("2d");
        var width = canvasElement.width;
        var height = canvasElement.height;

        var tileLength = 20;
        var textOffset = tileLength * 0.15;
        ctx.font = tileLength + 'px serif';
        ctx.lineWidth = 1;

        // draw unused tiles
        ctx.fillText("Unused Tiles:", 0, tileLength - textOffset);
        unusedTileIds = Object.keys(unusedTiles);
        for (var i = 0; i < unusedTileIds.length; i++) {
            var unusedTileId = unusedTileIds[i];
            this._drawTile(ctx, i * tileLength, tileLength, unusedTiles[unusedTileId], tileLength, textOffset);
        }

        // draw used grid
        ctx.fillText("Game Area:", 0, tileLength * 4 - textOffset);
        var usedPadding = 5;
        var tileLengthRound = x => Math.ceil(x / tileLength) * tileLength
        var usedMinX = tileLengthRound(usedPadding);
        var usedMinY = tileLengthRound(usedPadding + tileLength * 4);
        var usedMaxX = tileLengthRound(width - usedPadding);
        var usedMaxY = tileLengthRound(height - usedPadding);
        var numRows =  Math.floor((usedMaxY - usedMinY) / tileLength);
        var numCols = Math.floor((usedMaxX - usedMinX) / tileLength);
        // rows
        for (var i = 0; i <= numRows; i++) {
            ctx.moveTo(usedMinX, usedMinY + i * tileLength);
            ctx.lineTo(usedMaxX, usedMinY + i * tileLength);
        }
        // cols
        for (var i = 0; i <= numCols; i++) {
            ctx.moveTo(usedMinX + i * tileLength, usedMinY);
            ctx.lineTo(usedMinX + i * tileLength, usedMaxY);
        }

        ctx.stroke();
        console.log("done drawing");
    },

    _drawTile: function (ctx, x, y, tile, tileLength, textOffset) {
        ctx.strokeRect(x, y, tileLength, tileLength);
        ctx.fillText(tile.ch, x + textOffset, y + tileLength - textOffset);
    }
};