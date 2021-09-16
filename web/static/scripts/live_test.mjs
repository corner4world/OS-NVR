// Copyright 2020-2021 The OS-NVR Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { $ } from "./libs/common.mjs";
import { newViewer, resBtn } from "./live.mjs";

class mockHls {
	constructor() {}
	attachMedia() {}
	on() {}
	destroy() {}
}
mockHls.Events = {
	MEDIA_ATTACHED() {},
};

const monitors = {
	1: { enable: "false" },
	2: { enable: "true", id: "2" },
	3: { audioEnabled: "true", enable: "true", id: "3" },
};

describe("newViewer", () => {
	const setup = () => {
		document.body.innerHTML = `<div id="content-grid"></div>`;
		const element = $("#content-grid");
		const viewer = newViewer(element, monitors, mockHls);
		viewer.lowRes();
		viewer.highRes();
		viewer.reset();
		return element;
	};
	test("rendering", () => {
		const expected = `
			<div id="js-video-2 "class="grid-item-container">
				<video
					class="grid-item"
					muted=""
					disablepictureinpicture=""
				></video>
			</div>
			<div id="js-video-3" class="grid-item-container">
				<input
					class="player-overlay-checkbox"
					id="3-player-checkbox"
					type="checkbox"
				>
				<label 
					class="player-overlay-selector"
					for="3-player-checkbox"
				></label>
				<div class="player-overlay live-player-menu">
					<button class="live-player-btn js-mute-btn">
						<img
							class="icon"
							src="static/icons/feather/volume-x.svg"
						>
					</button>
				</div>
				<video
					class="grid-item"
					muted=""
					disablepictureinpicture=""
				></video>
			</div>`.replace(/\s/g, "");

		const element = setup();
		const actual = element.innerHTML.replace(/\s/g, "");

		expect(actual).toEqual(expected);
	});
	test("muteButton", () => {
		setup();
		const element = $("#js-video-3");
		const $video = element.querySelector("video");
		const $muteBtn = element.querySelector(".js-mute-btn");
		const $img = $muteBtn.querySelector("img");

		expect($video.muted).toBe(true);
		expect($img.src).toEqual("http://localhost/static/icons/feather/volume-x.svg");

		$muteBtn.click();
		expect($video.muted).toBe(false);
		expect($img.src).toEqual("http://localhost/static/icons/feather/volume.svg");

		$muteBtn.click();
		expect($video.muted).toBe(true);
		expect($img.src).toEqual("http://localhost/static/icons/feather/volume-x.svg");
	});
});

describe("resBtn", () => {
	const mockContent = {
		lowRes() {},
		highRes() {},
		reset() {},
	};
	test("working", () => {
		document.body.innerHTML = `<div></div>`;
		const element = $("div");

		const res = resBtn();
		element.innerHTML = res.html;

		const $btn = $(".js-res");
		expect($btn.textContent).toEqual("X");

		res.init(element, mockContent);
		expect($btn.textContent).toEqual("HD");

		$btn.click();
		expect($btn.textContent).toEqual("SD");
		expect(localStorage.getItem("highRes")).toEqual("false");

		$btn.click();
		expect($btn.textContent).toEqual("HD");
		expect(localStorage.getItem("highRes")).toEqual("true");

		$btn.click();
		expect($btn.textContent).toEqual("SD");
		expect(localStorage.getItem("highRes")).toEqual("false");
	});
	test("contentCalled", () => {
		document.body.innerHTML = `<div></div>`;
		const element = $("div");

		const res = resBtn();
		element.innerHTML = res.html;

		let low, high, reset;
		const content = {
			lowRes() {
				low = true;
			},
			highRes() {
				high = true;
			},
			reset() {
				reset = true;
			},
		};

		res.init(element, content);
		$(".js-res").click();
		expect(low).toEqual(true);
		expect(high).toEqual(true);
		expect(reset).toEqual(true);
	});
});
