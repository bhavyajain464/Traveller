function QRCodePreview({ value, size = 220 }) {
  const cells = buildMatrix(value, 21);
  const cellSize = size / cells.length;

  return (
    <svg
      aria-label="Travel QR code"
      className="qr-code"
      viewBox={`0 0 ${size} ${size}`}
      role="img"
    >
      <rect width={size} height={size} rx="18" fill="#fffdf8" />
      {cells.flatMap((row, y) =>
        row.map((filled, x) =>
          filled ? (
            <rect
              key={`${x}-${y}`}
              x={x * cellSize}
              y={y * cellSize}
              width={cellSize}
              height={cellSize}
              fill="#111827"
            />
          ) : null
        )
      )}
    </svg>
  );
}

function buildMatrix(value, dimension) {
  const matrix = Array.from({ length: dimension }, () => Array(dimension).fill(false));

  placeFinder(matrix, 0, 0);
  placeFinder(matrix, 0, dimension - 7);
  placeFinder(matrix, dimension - 7, 0);

  let seed = 0;
  for (const char of value) {
    seed = (seed * 31 + char.charCodeAt(0)) >>> 0;
  }

  for (let y = 0; y < dimension; y += 1) {
    for (let x = 0; x < dimension; x += 1) {
      if (isFinderZone(x, y, dimension)) {
        continue;
      }
      seed = (seed ^ (seed << 13)) >>> 0;
      seed = (seed ^ (seed >> 17)) >>> 0;
      seed = (seed ^ (seed << 5)) >>> 0;
      matrix[y][x] = (seed & 1) === 1;
    }
  }

  return matrix;
}

function placeFinder(matrix, startY, startX) {
  for (let y = 0; y < 7; y += 1) {
    for (let x = 0; x < 7; x += 1) {
      const edge = x === 0 || y === 0 || x === 6 || y === 6;
      const center = x >= 2 && x <= 4 && y >= 2 && y <= 4;
      matrix[startY + y][startX + x] = edge || center;
    }
  }
}

function isFinderZone(x, y, dimension) {
  const inTopLeft = x < 7 && y < 7;
  const inTopRight = x >= dimension - 7 && y < 7;
  const inBottomLeft = x < 7 && y >= dimension - 7;
  return inTopLeft || inTopRight || inBottomLeft;
}

export default QRCodePreview;
