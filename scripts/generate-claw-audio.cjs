const fs = require("node:fs");
const path = require("node:path");
let lamejs;

const sampleRate = 44100;
const outputDir = path.join(__dirname, "..", "static", "audio", "claw");
let randomState = 0x4d4b4741;

function random() {
  randomState = (1664525 * randomState + 1013904223) >>> 0;
  return randomState / 0x100000000;
}

function createSound(seconds) {
  return new Float64Array(Math.ceil(seconds * sampleRate));
}

function envelope(position, duration, attack = 0.01, release = 0.12) {
  const age = position / sampleRate;
  const length = duration / sampleRate;
  return Math.min(1, age / attack) * Math.min(1, Math.max(0, (length - age) / release));
}

function addTone(sound, start, duration, fromFrequency, toFrequency, volume, shape = "sine") {
  const first = Math.floor(start * sampleRate);
  const count = Math.floor(duration * sampleRate);
  let phase = 0;
  for (let index = 0; index < count && first + index < sound.length; index += 1) {
    const progress = index / count;
    const frequency = fromFrequency + (toFrequency - fromFrequency) * progress;
    phase += (Math.PI * 2 * frequency) / sampleRate;
    let wave = Math.sin(phase);
    if (shape === "triangle") wave = (2 / Math.PI) * Math.asin(wave);
    if (shape === "square") wave = wave < 0 ? -1 : 1;
    sound[first + index] += wave * volume * envelope(index, count);
  }
}

function addNoiseBurst(sound, start, duration, volume, brightness = 0.78) {
  const first = Math.floor(start * sampleRate);
  const count = Math.floor(duration * sampleRate);
  let previous = 0;
  for (let index = 0; index < count && first + index < sound.length; index += 1) {
    const raw = random() * 2 - 1;
    const bright = raw - previous * brightness;
    previous = raw;
    const decay = Math.exp(-7.5 * index / count);
    sound[first + index] += bright * volume * decay * Math.min(1, index / 80);
  }
}

function addClap(sound, time, volume) {
  addNoiseBurst(sound, time, 0.115, volume, 0.68);
  addNoiseBurst(sound, time + 0.019 + random() * 0.008, 0.09, volume * 0.72, 0.74);
  addNoiseBurst(sound, time + 0.047 + random() * 0.012, 0.07, volume * 0.46, 0.8);
  addTone(sound, time, 0.08, 185 + random() * 70, 125, volume * 0.16, "triangle");
}

function buildMotor() {
  const sound = createSound(1.05);
  for (let time = 0; time < 1.05; time += 0.075) {
    addTone(sound, time, 0.07, 74, 91, 0.12, "triangle");
    addNoiseBurst(sound, time, 0.035, 0.025, 0.35);
  }
  addTone(sound, 0, 1.05, 58, 63, 0.08, "triangle");
  addTone(sound, 0, 1.05, 116, 126, 0.035, "sine");
  return sound;
}

function buildGrab() {
  const sound = createSound(0.72);
  [0, 0.085, 0.18].forEach((time, index) => {
    addNoiseBurst(sound, time, 0.11, 0.26 - index * 0.04, 0.86);
    addTone(sound, time, 0.16, 390 - index * 55, 125, 0.18, "triangle");
    addTone(sound, time, 0.22, 1220 - index * 90, 760, 0.055, "sine");
  });
  return sound;
}

function buildDrop() {
  const sound = createSound(0.9);
  addTone(sound, 0, 0.48, 620, 105, 0.23, "triangle");
  addNoiseBurst(sound, 0.36, 0.16, 0.18, 0.52);
  addTone(sound, 0.42, 0.42, 880, 560, 0.13, "sine");
  addTone(sound, 0.45, 0.38, 1320, 920, 0.075, "sine");
  return sound;
}

function buildFail() {
  const sound = createSound(1.25);
  addTone(sound, 0, 0.42, 330, 245, 0.2, "triangle");
  addTone(sound, 0.38, 0.54, 245, 145, 0.22, "triangle");
  addNoiseBurst(sound, 0.86, 0.25, 0.07, 0.35);
  return sound;
}

function buildApplause() {
  const sound = createSound(4.8);
  const celebrationNotes = [523.25, 659.25, 783.99, 1046.5];
  celebrationNotes.forEach((frequency, index) => {
    addTone(sound, index * 0.115, 0.55, frequency, frequency * 1.01, 0.14, "triangle");
    addTone(sound, index * 0.115, 0.7, frequency * 2, frequency * 2.005, 0.045, "sine");
  });

  for (let person = 0; person < 22; person += 1) {
    let time = 0.22 + random() * 0.7;
    while (time < 4.45) {
      const fadeIn = Math.min(1, time / 0.85);
      const fadeOut = Math.min(1, (4.8 - time) / 0.75);
      addClap(sound, time, (0.055 + random() * 0.055) * fadeIn * fadeOut);
      time += 0.23 + random() * 0.48;
    }
  }

  for (let time = 0.35; time < 3.9; time += 0.28 + random() * 0.3) {
    addTone(sound, time, 0.32, 650 + random() * 320, 850 + random() * 420, 0.018, "sine");
  }
  return sound;
}

function encodeMp3(filename, samples) {
  let peak = 0;
  for (const sample of samples) peak = Math.max(peak, Math.abs(sample));
  const multiplier = peak > 0 ? 0.92 / peak : 1;
  const pcm = new Int16Array(samples.length);
  for (let index = 0; index < samples.length; index += 1) {
    const value = Math.tanh(samples[index] * multiplier * 1.35) / Math.tanh(1.35);
    pcm[index] = Math.max(-32768, Math.min(32767, Math.round(value * 32767)));
  }

  const encoder = new lamejs.Mp3Encoder(1, sampleRate, 160);
  const chunks = [];
  for (let index = 0; index < pcm.length; index += 1152) {
    const encoded = encoder.encodeBuffer(pcm.subarray(index, index + 1152));
    if (encoded.length) chunks.push(Buffer.from(encoded));
  }
  const finalChunk = encoder.flush();
  if (finalChunk.length) chunks.push(Buffer.from(finalChunk));
  fs.writeFileSync(path.join(outputDir, filename), Buffer.concat(chunks));
}

async function main() {
  const lameModule = await import("@breezystack/lamejs");
  lamejs = lameModule.default || lameModule;
  fs.mkdirSync(outputDir, { recursive: true });
  encodeMp3("motor.mp3", buildMotor());
  encodeMp3("grab.mp3", buildGrab());
  encodeMp3("drop.mp3", buildDrop());
  encodeMp3("fail.mp3", buildFail());
  encodeMp3("win-applause.mp3", buildApplause());
  console.log(`Generated claw audio in ${outputDir}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
