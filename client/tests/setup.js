import { JSDOM } from "jsdom";

const dom = new JSDOM("<!DOCTYPE html><html><body></body></html>", {
	url: "http://localhost",
});

global.window = dom.window;

for (const key in dom.window) {
	if (!(key in global)) {
		global[key] = dom.window[key];
	}
}
