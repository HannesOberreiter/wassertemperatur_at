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

type Data = Badegewaesser & {
  BUNDESLAND?: string;
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

const SOURCE_URL = "https://www.ages.at/typo3temp/badegewaesser_db.json";

export const fetchData = async (): Promise<Data[]> => {
  const lakes = await fetch(SOURCE_URL);
  const result = (await lakes.json()) as LakesData;
  const data: Data[] = [];

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

  return data;
};
