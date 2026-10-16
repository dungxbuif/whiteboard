// Ordinary JS heap allocations exercise V8's production worker heap cap.
const retained = [];
while (true) retained.push(new Array(100_000).fill('retained'));
