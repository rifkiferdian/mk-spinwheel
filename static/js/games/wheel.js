const app = document.querySelector("#wheel-app");

if (app) {
  const slug = app.dataset.campaignSlug;
  const svgNS = "http://www.w3.org/2000/svg";
  const wheelSVG = document.querySelector("#wheel-svg");
  const wheelLights = document.querySelector("#wheel-lights");
  const wheelRotor = document.querySelector("#wheel-rotor");
  const wheelGroup = document.querySelector("#wheel-segments");
  const pointerTip = document.querySelector("#wheel-pointer-tip");
  const spinButton = document.querySelector("#spin-button");
  const spinButtonText = document.querySelector("#spin-button-text");
  const statusText = document.querySelector("#game-status");
  const headline = document.querySelector("#game-headline");
  const dialog = document.querySelector("#result-dialog");
  const resultTitle = document.querySelector("#result-title");
  const resultDescription = document.querySelector("#result-description");
  const claimBox = document.querySelector("#claim-box");
  const claimCode = document.querySelector("#claim-code");
  const playAgainButton = document.querySelector("#play-again-button");
  let campaign;
  let rotation = 0;
  let spinning = false;

  const polar = (angle, radius) => {
    const radians = (angle * Math.PI) / 180;
    return { x: 300 + radius * Math.cos(radians), y: 300 + radius * Math.sin(radians) };
  };

  const segmentPath = (start, end) => {
    const first = polar(start, 280);
    const second = polar(end, 280);
    const largeArc = end - start > 180 ? 1 : 0;
    return `M 300 300 L ${first.x} ${first.y} A 280 280 0 ${largeArc} 1 ${second.x} ${second.y} Z`;
  };

  const splitLabel = (name) => {
    if (name.length <= 24) return [name];
    const words = name.split(" ");
    const midpoint = Math.ceil(words.length / 2);
    return [words.slice(0, midpoint).join(" "), words.slice(midpoint).join(" ")];
  };

  function drawWheel(prizes) {
    wheelGroup.replaceChildren();
    const angle = 360 / prizes.length;
    prizes.forEach((prize, index) => {
      const start = -90 + index * angle;
      const end = start + angle;
      const center = start + angle / 2;
      const path = document.createElementNS(svgNS, "path");
      path.setAttribute("d", segmentPath(start, end));
      path.setAttribute("fill", prize.color || "#6366f1");
      path.setAttribute("stroke", "rgba(255,255,255,.75)");
      path.setAttribute("stroke-width", "4");
      path.dataset.prizeId = prize.id;
      wheelGroup.append(path);

      const point = polar(center, prizes.length > 6 ? 175 : 168);
      const text = document.createElementNS(svgNS, "text");
      text.setAttribute("x", point.x);
      text.setAttribute("y", point.y);
      text.setAttribute("text-anchor", "middle");
      text.setAttribute("dominant-baseline", "middle");
      text.setAttribute("fill", "#ffffff");
      text.setAttribute("font-size", prizes.length > 6 ? "16" : "19");
      text.setAttribute("font-weight", "800");
      text.setAttribute("paint-order", "stroke");
      text.setAttribute("stroke", "rgba(15,23,42,.28)");
      text.setAttribute("stroke-width", "3");
      // Label mengikuti jari-jari roda. Label pada separuh kiri dibalik agar
      // tetap terbaca saat diam. Ketika segmen berhenti pada pointer, hasilnya
      // selalu berorientasi vertikal ±90°, bukan terbalik 180°.
      const normalizedCenter = ((center % 360) + 360) % 360;
      const labelRotation = normalizedCenter > 90 && normalizedCenter < 270 ? center + 180 : center;
      text.setAttribute("transform", `rotate(${labelRotation} ${point.x} ${point.y})`);
      splitLabel(prize.name).forEach((line, lineIndex, lines) => {
        const span = document.createElementNS(svgNS, "tspan");
        span.setAttribute("x", point.x);
        span.setAttribute("dy", lineIndex === 0 ? (lines.length > 1 ? "-0.55em" : "0") : "1.15em");
        span.textContent = line;
        text.append(span);
      });
      wheelGroup.append(text);
    });
  }

  function drawLights() {
    wheelLights.replaceChildren();
    const totalLights = 32;
    for (let index = 0; index < totalLights; index += 1) {
      const point = polar(-90 + (index * 360) / totalLights, 286);
      const light = document.createElementNS(svgNS, "circle");
      light.setAttribute("cx", point.x);
      light.setAttribute("cy", point.y);
      light.setAttribute("r", index % 2 === 0 ? "5.5" : "4.5");
      light.setAttribute("class", `wheel-light wheel-light-phase-${index % 4}`);
      wheelLights.append(light);
    }
  }

  async function request(path, body) {
    const response = await fetch(path, {
      method: body ? "POST" : "GET",
      headers: body ? { "Content-Type": "application/json" } : {},
      body: body ? JSON.stringify(body) : undefined,
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || "Permainan tidak dapat diproses");
    return data;
  }

  async function initialize() {
    try {
      campaign = await request(`/api/campaign/${encodeURIComponent(slug)}`);
      drawLights();
      drawWheel(campaign.prizes);
      if (campaign.config?.headline) headline.textContent = campaign.config.headline;
      spinButton.disabled = false;
      spinButtonText.textContent = "PUTAR SEKARANG";
      statusText.textContent = "Roda siap. Semoga beruntung!";
    } catch (error) {
      spinButtonText.textContent = "PERMAINAN TIDAK TERSEDIA";
      statusText.textContent = error.message;
    }
  }

  async function spin() {
    if (spinning || !campaign) return;
    spinning = true;
    wheelSVG.classList.add("wheel-is-spinning");
    spinButton.disabled = true;
    spinButtonText.textContent = "MENGACAK HADIAH…";
    statusText.textContent = "Keberuntungan sedang memilih hadiah Anda.";
    try {
      const session = await request("/api/game/session", { campaignSlug: campaign.slug });
      const result = await request("/api/game/play", { sessionToken: session.sessionToken });
      const prizeIndex = campaign.prizes.findIndex((prize) => prize.id === result.prizeId);
      if (prizeIndex < 0) throw new Error("Hadiah tidak ditemukan pada roda");
      const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      const duration = reducedMotion ? 900 : Number(campaign.config?.duration_ms || 6000);
      await animateWheel(prizeIndex, duration);
      wheelSVG.classList.remove("wheel-is-spinning");
      highlightPrize(result.prizeId);
      statusText.textContent = `Hasil: ${result.prizeName}`;
      await new Promise((resolve) => window.setTimeout(resolve, reducedMotion ? 100 : 600));
      showResult(result);
    } catch (error) {
      statusText.textContent = error.message;
      spinButton.disabled = false;
      spinButtonText.textContent = "COBA LAGI";
      spinning = false;
      wheelSVG.classList.remove("wheel-is-spinning");
    }
  }

  function normalizeAngle(value) {
    return ((value % 360) + 360) % 360;
  }

  function setWheelRotation(value) {
    // Gunakan transform SVG native dengan poros eksplisit. Jangan memakai
    // CSS transform karena referensinya dapat berubah mengikuti bounding box.
    wheelRotor.setAttributeNS(null, "transform", `rotate(${value} 300 300)`);
  }

  function animateValue(from, to, duration, easing, onFrame) {
    return new Promise((resolve) => {
      const started = performance.now();
      function frame(now) {
        const progress = Math.min(1, (now - started) / duration);
        const value = from + (to - from) * easing(progress);
        onFrame(value, progress);
        if (progress < 1) requestAnimationFrame(frame);
        else resolve();
      }
      requestAnimationFrame(frame);
    });
  }

  function spinEasing(progress) {
    if (progress < 0.18) {
      const phase = progress / 0.18;
      return 0.09 * phase * phase;
    }
    const phase = (progress - 0.18) / 0.82;
    return 0.09 + 0.91 * (1 - Math.pow(1 - phase, 4.5));
  }

  function pointerTick() {
    if (!pointerTip.getAnimations().length) {
      pointerTip.animate(
        [
          { transform: "rotate(0deg)" },
          { transform: "rotate(-16deg)", offset: 0.42 },
          { transform: "rotate(5deg)", offset: 0.76 },
          { transform: "rotate(0deg)" },
        ],
        { duration: 115, easing: "ease-out" },
      );
    }
  }

  async function animateWheel(prizeIndex, duration) {
    const segmentAngle = 360 / campaign.prizes.length;
    const current = normalizeAngle(rotation);
    const desired = normalizeAngle(360 - (prizeIndex + 0.5) * segmentAngle);
    const adjustment = normalizeAngle(desired - current);
    const turns = duration < 1500 ? 2 : 7;
    const windup = rotation - (duration < 1500 ? 3 : 9);
    let previousBoundary = Math.floor(rotation / segmentAngle);
    let lastTickAt = 0;

    await animateValue(rotation, windup, Math.min(260, duration * 0.12), (t) => 1 - Math.pow(1 - t, 3), setWheelRotation);

    const target = rotation + turns * 360 + adjustment;
    await animateValue(windup, target, duration, spinEasing, (value) => {
      setWheelRotation(value);
      const boundary = Math.floor(value / segmentAngle);
      const now = performance.now();
      if (boundary !== previousBoundary && now - lastTickAt > 68) {
        pointerTick();
        previousBoundary = boundary;
        lastTickAt = now;
      }
    });

    if (duration >= 1500) {
      await animateValue(target, target + 1.8, 110, (t) => 1 - Math.pow(1 - t, 2), setWheelRotation);
      await animateValue(target + 1.8, target, 180, (t) => 1 - Math.pow(1 - t, 3), setWheelRotation);
    }
    rotation = desired;
    setWheelRotation(rotation);
  }

  function highlightPrize(prizeId) {
    const selected = wheelGroup.querySelector(`[data-prize-id="${prizeId}"]`);
    if (!selected) return;
    selected.animate(
      [
        { opacity: 1, filter: "brightness(1)" },
        { opacity: 0.72, filter: "brightness(1.4)", offset: 0.35 },
        { opacity: 1, filter: "brightness(1.12)" },
      ],
      { duration: 750, iterations: 2, easing: "ease-in-out" },
    );
  }

  function showResult(result) {
    resultTitle.textContent = result.claimStatus === "not_required" ? result.prizeName : `Anda mendapat ${result.prizeName}!`;
    resultDescription.textContent = result.prizeDescription || "Terima kasih sudah bermain.";
    claimBox.hidden = !result.claimCode;
    claimCode.textContent = result.claimCode || "";
    dialog.showModal();
    if (result.claimStatus !== "not_required") launchConfetti();
  }

  spinButton.addEventListener("click", spin);
  playAgainButton.addEventListener("click", () => {
    dialog.close();
    spinning = false;
    spinButton.disabled = false;
    spinButtonText.textContent = "PUTAR LAGI";
    statusText.textContent = "Roda siap untuk permainan berikutnya.";
  });

  initialize();
}

function launchConfetti() {
  const canvas = document.querySelector("#confetti-canvas");
  const context = canvas.getContext("2d");
  const colors = ["#fbbf24", "#f97316", "#6366f1", "#ec4899", "#22c55e"];
  canvas.width = window.innerWidth;
  canvas.height = window.innerHeight;
  const particles = Array.from({ length: 130 }, () => ({
    x: Math.random() * canvas.width,
    y: -20 - Math.random() * canvas.height * 0.35,
    size: 5 + Math.random() * 8,
    speed: 2 + Math.random() * 5,
    drift: -2 + Math.random() * 4,
    rotation: Math.random() * Math.PI,
    color: colors[Math.floor(Math.random() * colors.length)],
  }));
  const started = performance.now();
  function frame(now) {
    context.clearRect(0, 0, canvas.width, canvas.height);
    particles.forEach((particle) => {
      particle.y += particle.speed;
      particle.x += particle.drift;
      particle.rotation += 0.08;
      context.save();
      context.translate(particle.x, particle.y);
      context.rotate(particle.rotation);
      context.fillStyle = particle.color;
      context.fillRect(-particle.size / 2, -particle.size / 3, particle.size, particle.size / 1.5);
      context.restore();
    });
    if (now - started < 3500) requestAnimationFrame(frame);
    else context.clearRect(0, 0, canvas.width, canvas.height);
  }
  requestAnimationFrame(frame);
}
