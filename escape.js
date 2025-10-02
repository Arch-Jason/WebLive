import * as fs from "fs";

const str = String(fs.readFileSync("./index.html"));

console.log(JSON.stringify(str));
