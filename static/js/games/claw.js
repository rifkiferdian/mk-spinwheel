const clawApp = document.querySelector("#claw-app");

if (clawApp) {
  const slug = clawApp.dataset.campaignSlug;
  const machineWindow = document.querySelector("#claw-window");
  const carriage = document.querySelector("#claw-carriage");
  const line = document.querySelector("#claw-line");
  const grip = document.querySelector("#claw-grip");
  const caughtPrize = document.querySelector("#claw-catch");
  const prizeField = document.querySelector("#claw-prizes");
  const chute = document.querySelector("#claw-chute");
  const chuteLabel = document.querySelector("#claw-chute-label");
  const chutePrize = document.querySelector("#claw-chute-prize");
  const button = document.querySelector("#claw-button");
  const buttonText = document.querySelector("#claw-button-text");
  const statusText = document.querySelector("#claw-status");
  const screenIcon = document.querySelector("#claw-screen-icon");
  const headline = document.querySelector("#claw-headline");
  const dialog = document.querySelector("#claw-result-dialog");
  const resultTitle = document.querySelector("#claw-result-title");
  const resultDescription = document.querySelector("#claw-result-description");
  const claimBox = document.querySelector("#claw-claim-box");
  const claimCode = document.querySelector("#claw-claim-code");
  const playAgain = document.querySelector("#claw-play-again");
  let campaign;
  let playing = false;
  let carriageOffset = 0;
  let lineHeight = 55;

  const wait = (duration) => new Promise((resolve) => window.setTimeout(resolve, duration));

  function emojiForPrize(name) {
    const normalized = name.toLowerCase();
    if (normalized.includes("beruang")) return "🧸";
    if (normalized.includes("panda")) return "🐼";
    if (normalized.includes("kelinci")) return "🐰";
    if (normalized.includes("voucher")) return "🎟️";
    return "❔";
  }

  function renderPrizes() {
    prizeField.replaceChildren();
    const visualPrizes = [];
    let remaining = campaign.prizes.map((prize) => Math.max(1, Number(prize.visualCount || 1)));
    while (remaining.some((count) => count > 0)) {
      campaign.prizes.forEach((prize, index) => {
        if (remaining[index] > 0) {
          visualPrizes.push(prize);
          remaining[index] -= 1;
        }
      });
    }
    visualPrizes.forEach((prize, index) => {
      const item = document.createElement("div");
      item.className = `claw-prize claw-pile-${index % 20}`;
      item.dataset.prizeId = prize.id;
      item.title = prize.name;
      const emoji = document.createElement("span");
      emoji.textContent = emojiForPrize(prize.name);
      item.append(emoji);
      prizeField.append(item);
    });
  }

  async function request(path, body) {
    const response = await fetch(path, {
      method: body ? "POST" : "GET",
      headers: body ? { "Content-Type": "application/json" } : {},
      body: body ? JSON.stringify(body) : undefined,
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Mesin tidak dapat diproses");
    return data;
  }

  async function initialize() {
    try {
      campaign = await request(`/api/campaign/${encodeURIComponent(slug)}`);
      renderPrizes();
      if (campaign.config?.headline) headline.textContent = campaign.config.headline;
      button.disabled = false;
      buttonText.textContent = "CAPIT SEKARANG";
      statusText.textContent = "Mesin siap. Pilih keberuntungan Anda!";
      screenIcon.textContent = "✓";
    } catch (error) {
      buttonText.textContent = "GAME TIDAK TERSEDIA";
      statusText.textContent = error.message;
      screenIcon.textContent = "!";
    }
  }

  function moveCarriage(targetOffset, duration) {
    const animation = carriage.animate(
      [
        { transform: `translateX(-50%) translateX(${carriageOffset}px)` },
        { transform: `translateX(-50%) translateX(${targetOffset}px)` },
      ],
      { duration, easing: "cubic-bezier(.25,.75,.25,1)", fill: "forwards" },
    );
    carriageOffset = targetOffset;
    return animation.finished;
  }

  function moveLine(targetHeight, duration) {
    const animation = line.animate(
      [{ height: `${lineHeight}px` }, { height: `${targetHeight}px` }],
      { duration, easing: "cubic-bezier(.35,.05,.2,1)", fill: "forwards" },
    );
    lineHeight = targetHeight;
    return animation.finished;
  }

  async function animateClaw(prizeIndex, result) {
    const width = machineWindow.clientWidth;
    const targetPrize = prizeField.querySelector(`[data-prize-id="${result.prizeId}"]`);
    const windowBounds = machineWindow.getBoundingClientRect();
    const prizeBounds = targetPrize?.getBoundingClientRect();
    const fallbackPosition = (prizeIndex + 1) / (campaign.prizes.length + 1);
    const targetCenter = prizeBounds ? prizeBounds.left + prizeBounds.width / 2 : windowBounds.left + fallbackPosition * width;
    const targetOffset = targetCenter - (windowBounds.left + width / 2);
    const targetCenterY = prizeBounds ? prizeBounds.top + prizeBounds.height / 2 : windowBounds.bottom - 95;
    const dropHeight = Math.max(110, targetCenterY - windowBounds.top - 82);
    const chuteOffset = -0.38 * width;
    const won = result.claimStatus !== "not_required";

    statusText.textContent = "Capit bergerak menuju hadiah…";
    await moveCarriage(targetOffset, 850);
    statusText.textContent = "Capit diturunkan…";
    await moveLine(dropHeight, 1150);
    grip.classList.add("claw-closed");
    await wait(430);

    if (won) {
      caughtPrize.textContent = emojiForPrize(result.prizeName);
      caughtPrize.hidden = false;
      targetPrize?.classList.add("is-captured");
      statusText.textContent = "Hadiah berhasil dijepit!";
      screenIcon.textContent = "★";
    } else {
      statusText.textContent = "Capit belum mendapatkan hadiah…";
      screenIcon.textContent = "×";
    }

    await moveLine(55, 950);
    if (won) {
      chute.classList.add("is-active");
      chuteLabel.textContent = "BERSIAP MENERIMA";
      await moveCarriage(chuteOffset, 900);
      const chuteBounds = chute.getBoundingClientRect();
      const chuteLineHeight = Math.max(120, chuteBounds.top - windowBounds.top - 82);
      await moveLine(chuteLineHeight, 620);
      grip.classList.remove("claw-closed");
      const drop = caughtPrize.animate(
        [{ transform: "translate(-50%,0) scale(1)", opacity: 1 }, { transform: "translate(-50%,58px) rotate(22deg) scale(.72)", opacity: 0 }],
        { duration: 540, easing: "cubic-bezier(.25,.75,.35,1)", fill: "forwards" },
      );
      await drop.finished;
      chutePrize.textContent = emojiForPrize(result.prizeName);
      chutePrize.hidden = false;
      chuteLabel.textContent = "HADIAH BERHASIL MASUK!";
      chute.classList.add("is-delivered");
      statusText.textContent = "Hadiah masuk ke tempat pengambilan!";
      await wait(950);
    } else {
      const shake = carriage.animate(
        [
          { transform: `translateX(-50%) translateX(${targetOffset}px)` },
          { transform: `translateX(-50%) translateX(${targetOffset - 8}px)` },
          { transform: `translateX(-50%) translateX(${targetOffset + 8}px)` },
          { transform: `translateX(-50%) translateX(${targetOffset}px)` },
        ],
        { duration: 380, iterations: 2 },
      );
      await shake.finished;
    }
  }

  async function play() {
    if (playing || !campaign) return;
    playing = true;
    button.disabled = true;
    buttonText.textContent = "CAPIT SEDANG BERGERAK…";
    screenIcon.textContent = "…";
    try {
      const session = await request("/api/game/session", { campaignSlug: campaign.slug });
      const result = await request("/api/game/play", { sessionToken: session.sessionToken });
      const prizeIndex = campaign.prizes.findIndex((prize) => prize.id === result.prizeId);
      if (prizeIndex < 0) throw new Error("Hadiah tidak ditemukan dalam mesin");
      await animateClaw(prizeIndex, result);
      showResult(result);
    } catch (error) {
      statusText.textContent = error.message;
      screenIcon.textContent = "!";
      button.disabled = false;
      buttonText.textContent = "COBA LAGI";
      playing = false;
    }
  }

  function showResult(result) {
    resultTitle.textContent = result.claimStatus === "not_required" ? result.prizeName : `Anda mendapat ${result.prizeName}!`;
    resultDescription.textContent = result.prizeDescription || "Terima kasih sudah bermain.";
    claimBox.hidden = !result.claimCode;
    claimCode.textContent = result.claimCode || "";
    dialog.showModal();
    playAgain.focus();
    if (result.claimStatus !== "not_required") launchClawConfetti();
  }

  function resetMachine() {
    [...carriage.getAnimations(), ...line.getAnimations(), ...caughtPrize.getAnimations()].forEach((animation) => animation.cancel());
    carriageOffset = 0;
    lineHeight = 55;
    grip.classList.remove("claw-closed");
    caughtPrize.hidden = true;
    chute.classList.remove("is-active", "is-delivered");
    chutePrize.hidden = true;
    chuteLabel.textContent = "AMBIL HADIAH";
    renderPrizes();
    screenIcon.textContent = "✓";
    statusText.textContent = "Mesin siap untuk permainan berikutnya.";
    button.disabled = false;
    buttonText.textContent = "CAPIT LAGI";
    playing = false;
  }

  button.addEventListener("click", play);
  playAgain.addEventListener("click", () => { dialog.close(); resetMachine(); });
  document.addEventListener("keydown", (event) => {
    if (event.key !== "Enter" || event.repeat) return;

    event.preventDefault();
    if (dialog.open) {
      playAgain.click();
      return;
    }

    if (!playing && campaign && !button.disabled) button.click();
  });
  initialize();
}

function launchClawConfetti() {
  const canvas = document.querySelector("#claw-confetti");
  const context = canvas.getContext("2d");
  const colors = ["#f97316", "#facc15", "#292524", "#fb923c", "#ffffff"];
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
  const particles = Array.from({ length: 120 }, () => ({ x: Math.random()*canvas.width, y:-20-Math.random()*240, size:5+Math.random()*8, speed:2+Math.random()*5, drift:-2+Math.random()*4, color:colors[Math.floor(Math.random()*colors.length)] }));
  const started = performance.now();
  function frame(now) {
    context.clearRect(0,0,canvas.width,canvas.height);
    particles.forEach((particle) => { particle.y+=particle.speed;particle.x+=particle.drift;context.fillStyle=particle.color;context.fillRect(particle.x,particle.y,particle.size,particle.size*.65); });
    if(now-started<3200)requestAnimationFrame(frame);else context.clearRect(0,0,canvas.width,canvas.height);
  }
  requestAnimationFrame(frame);
}
