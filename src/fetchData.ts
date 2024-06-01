import { load } from "cheerio";

type LakesData = {
  VERSION: string;
  BUNDESLAENDER: Bundeslaender[];
};

type Bundeslaender = {
  BUNDESLAND: string;
  BADEGEWAESSER: Badegewaesser[];
};

type Badegewaesser = {
  BADEGEWAESSERID: string;
  BADEGEWAESSERNAME: string;
  BEZIRK: string;
  GEMEINDE: string;
  LONGITUDE: string;
  LATITUDE: string;
  ANSPRECHSTELLE: string;
  STRASSE_NUMMER: string;
  PLZ_ORT: string;
  TELEFON: string;
  EMAIL: string;
  TGESPERRT: string;
  ENTEROKOKKEN_EINHEIT: string;
  E_COLI_EINHEIT: string;
  WASSERTEMPERATUR_EINHEIT: string;
  SICHTTIEFE_EINHEIT: string;
  SPERRGRUND: string;
  WASSERQUALITAET_JAHR_HEUER: string;
  QUALITAET_2023: string;
  WASSERQUALITAET_JAHR_VORIGES: string;
  QUALITAET_2022: string;
  WASSERQUALITAET_JAHR_VOR_VORIGES: string;
  QUALITAET_2021: string;
  WASSERQUALITAET_JAHR_VOR_VOR_VORIGES: string;
  QUALITAET_2020: string;
  QUALITAET_2019: string;
  MESSWERTE: Array<{
    D: string;
    O_E: string | null;
    E: number;
    O_EC: string | null;
    E_C: number;
    W: number;
    S: number;
    A: number;
  }>;
};

export type Data = Badegewaesser & {
  BUNDESLAND?: string;
};

export type Waters = {
  BUNDESLAND: string;
  AGES: boolean;
  NAME: string;
  DATUM: string;
  RECENT: boolean;
  TEMPERATUR: number | undefined;
  SICHTTIEFE: number | undefined;
  QUALITÄT: number | undefined;
  data?: Array<{ timestamp: string; value: number }>;
};

export const STATES = {
  Oberösterreich: "Oberösterreich",
  Niederösterreich: "Niederösterreich",
  Wien: "Wien",
  Burgenland: "Burgenland",
  Steiermark: "Steiermark",
  Kärnten: "Kärnten",
  Salzburg: "Salzburg",
  Tirol: "Tirol",
  Vorarlberg: "Vorarlberg",
} as const;

const VERCEL_TIMEOUT = 9000;

// https://www.data.gv.at/katalog/dataset/2646025c-8ab9-4850-b76c-c2f508b34798
const URL_AGES = "https://www.ages.at/typo3temp/badegewaesser_db.json";
// https://www.data.gv.at/katalog/dataset/586e08a5-1ca6-400b-b2e2-dfd8fd3f429d
const URL_OÖ = "https://data.ooe.gv.at/files/hydro/HDOOE_Export_WT.zrxp";

const cacheTable = new Map<string, Waters[]>();
const cacheAGES = new Map<string, Data[]>();
const cacheTimestamp = new Map<string, number>();

export async function fetchDataAGES(): Promise<Data[]> {
  console.log("test");
  if (cacheAGES.has("data") && cacheTimestamp.has("data")) {
    const timestamp = cacheTimestamp.get("data") as number;
    const now = Date.now();
    const diff = now - timestamp;
    if (diff > 1000 * 60 * 60 * 5) {
      cacheAGES.delete("data");
      cacheTimestamp.delete("data");
    } else {
      return cacheAGES.get("data") as Data[];
    }
  }

  const timeout = setTimeout(() => {
    throw new Error("fetchDataAGES took too long");
  }, VERCEL_TIMEOUT);

  const lakes = await fetch(URL_AGES);

  const result = (await lakes.json()) as LakesData;
  const data: Data[] = [];

  clearTimeout(timeout);

  for (let i = 0; i < result.BUNDESLAENDER.length; i++) {
    const bundesland = result.BUNDESLAENDER[i];
    for (let j = 0; j < bundesland.BADEGEWAESSER.length; j++) {
      const badegewaesser = bundesland.BADEGEWAESSER[j] as Data;
      badegewaesser["BUNDESLAND"] = bundesland.BUNDESLAND;
      data.push(badegewaesser);
    }
  }
  data.sort((a, b) => {
    if (a.BADEGEWAESSERNAME < b.BADEGEWAESSERNAME) return -1;
    if (a.BADEGEWAESSERNAME > b.BADEGEWAESSERNAME) return 1;
    return 0;
  });

  cacheAGES.set("data", data);
  cacheTimestamp.set("data", Date.now());
  return data;
}

export async function tableData(): Promise<Waters[]> {
  if (cacheTable.has("tableData") && cacheTimestamp.has("tableData")) {
    const timestamp = cacheTimestamp.get("tableData") as number;
    const now = Date.now();
    const diff = now - timestamp;
    if (diff > 1000 * 60 * 60 * 5) {
      console.log("chartData from cache is too old");
      cacheTable.delete("tableData");
      cacheTimestamp.delete("tableData");
    } else {
      return cacheTable.get("tableData") as Waters[];
    }
  }
  let waters: Waters[] = [];

  const fetchAGES = fetch(URL_AGES);
  const fetchOÖ = fetch(URL_OÖ);
  const fetchAusseerlandPromise = fetchAusseerland();

  const [responseAGES, responseOÖ, responseAusseerland] =
    await Promise.allSettled([fetchAGES, fetchOÖ, fetchAusseerlandPromise]);

  if (responseAGES.status === "rejected") console.error(responseAGES.reason);
  if (responseOÖ.status === "rejected") console.error(responseOÖ.reason);

  if (responseAGES.status === "fulfilled") {
    const resultAGES = (await responseAGES.value?.json()) as LakesData;
    for (let i = 0; i < resultAGES.BUNDESLAENDER.length; i++) {
      const bundesland = resultAGES.BUNDESLAENDER[i];
      for (let j = 0; j < bundesland.BADEGEWAESSER.length; j++) {
        const badegewaesser = bundesland.BADEGEWAESSER[j] as Data;
        const date = badegewaesser.MESSWERTE[0].D.split(".");
        const newDate = `${date[2]}-${date[1]}-${date[0]}`;
        waters.push({
          BUNDESLAND: bundesland.BUNDESLAND,
          AGES: true,
          NAME: badegewaesser.BADEGEWAESSERNAME,
          DATUM: newDate,
          TEMPERATUR: badegewaesser.MESSWERTE[0].W,
          SICHTTIEFE: badegewaesser.MESSWERTE[0].S,
          QUALITÄT: badegewaesser.MESSWERTE[0].A,
          RECENT: dateIsRecent(newDate),
        });
      }
    }
  }

  if (responseOÖ.status === "fulfilled") {
    const resultOÖ = await responseOÖ.value.arrayBuffer();
    const decoder = new TextDecoder("ISO-8859-1");
    const decoded = decoder.decode(resultOÖ);
    const data = parseZRXP(decoded);
    waters = waters.concat(data);
  }

  if (responseAusseerland.status === "fulfilled") {
    const data = responseAusseerland.value;
    waters = waters.concat(data);
  }

  waters.sort((a, b) => {
    if (a.NAME < b.NAME) return -1;
    if (a.NAME > b.NAME) return 1;
    return 0;
  });

  cacheTable.set("tableData", waters);
  cacheTimestamp.set("tableData", Date.now());
  return waters;
}

export async function timeoutWrapper(fn: Function): Promise<Waters[] | Data[]> {
  return await new Promise(async (resolve, _reject) => {
    const timeout = setTimeout(() => {
      console.error("Server function took too long");
      resolve([]);
    }, VERCEL_TIMEOUT);

    const result = await fn();

    clearTimeout(timeout);

    resolve(result);
  });
}

function parseZRXP(content: string) {
  const metaRegrex = /SNAME(.*?)\|.*?SWATER(.*?)\|.*?CNAME(.*?)\|/;
  const lines = content.split("\n");
  const data: Waters[] = [];

  let i = 0;
  for (const line of lines) {
    if (line.startsWith("#SANR")) {
      const matches = line.match(metaRegrex);

      if (matches && matches.length > 3) {
        i++;
        data[i] = {
          BUNDESLAND: "Oberösterreich",
          NAME: matches[2] + ", " + matches[1],
          AGES: false,
          SICHTTIEFE: undefined,
          QUALITÄT: undefined,
          TEMPERATUR: undefined,
          DATUM: "",
          data: [],
          RECENT: false,
        };
        if (i > 1) {
          let previous = data[i - 1];
          if (previous.data && previous.data.length > 0) {
            previous.TEMPERATUR = previous.data[previous.data.length - 1].value;
            previous.DATUM = previous.data[
              previous.data.length - 1
            ].timestamp.replace(
              /(\d{4})(\d{2})(\d{2})(\d{2})(\d{2})(\d{2})/,
              "$1-$2-$3 $4:$5"
            );
            previous.RECENT = dateIsRecent(previous.DATUM);
          }
        }
      }
    }
    if (line.startsWith("#")) continue;
    const [timestamp, val] = line.split(" ");
    if (parseFloat(val) < -50 || parseFloat(val) > 50) continue;
    if (data[i].data) {
      // @ts-ignore
      data[i].data.push({ timestamp, value: parseFloat(val) });
    }
  }

  return data;
}

/**
 * @description E-Mail from 2024-31-05, Tourismusverband Ausseerland Salzkammergut, there is no API for this data. They manually insert the data around 3 times a week based on reported data (Montag, Mittwoch, Freitag). Additionally they send the data to Bergfex.
 */
async function fetchAusseerland() {
  try {
    const url =
      "https://www.steiermark.com/de/Ausseerland-Salzkammergut/Region/Sommerfrische/Seen-im-Ausseerland/Wassertemperaturen";

    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const html = await response.text();
    const $ = load(html);

    /* The date is in the last table header */
    const tableHead = $("table thead tr th").last().text().trim();
    const dateMatch = tableHead.match(/\((.*?)\)/);
    const date = dateMatch ? dateMatch[1] : "";
    const isoDate = date.replace(/(\d{2})\.(\d{2})\.(\d{4})/, "$3-$2-$1");

    const tableRows = $("table tbody tr");
    const data: Waters[] = [];

    tableRows.each((_index, row) => {
      const gewasser = $(row).find("td").first().text().trim();
      const temperatur = $(row).find("td").last().text().trim();

      data.push({
        BUNDESLAND: "Steiermark",
        NAME: gewasser,
        AGES: false,
        SICHTTIEFE: undefined,
        QUALITÄT: undefined,
        TEMPERATUR: parseFloat(temperatur),
        DATUM: isoDate,
        data: [],
        RECENT: dateIsRecent(isoDate),
      });
    });

    return data;
  } catch (error) {
    console.error(error);
    return [];
  }
}

/**
 * @description Check if the date is recent, not older than 2 weeks
 * @param date string ISO 8601 date
 */
function dateIsRecent(date: string) {
  const now = new Date();
  const dateObj = new Date(date);
  const diff = now.getTime() - dateObj.getTime();
  return diff < 1000 * 60 * 60 * 24 * 14;
}
