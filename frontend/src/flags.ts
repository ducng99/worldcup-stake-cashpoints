// Maps TLA codes and exact team names (from db/seed.go) to flag-icons ISO2 codes.
const TLA_MAP: Record<string, string> = {
  // UEFA
  FRA: 'fr', GER: 'de', ESP: 'es', POR: 'pt',
  ENG: 'gb-eng', NED: 'nl', BEL: 'be', CRO: 'hr',
  SUI: 'ch', BIH: 'ba', CZE: 'cz', AUT: 'at',
  SCO: 'gb-sct', TUR: 'tr', NOR: 'no', SWE: 'se',
  // CONMEBOL
  ARG: 'ar', BRA: 'br', COL: 'co', ECU: 'ec',
  URY: 'uy', PAR: 'py',
  // CONCACAF
  USA: 'us', MEX: 'mx', CAN: 'ca', PAN: 'pa',
  HAI: 'ht', CUW: 'cw',
  // CAF
  MAR: 'ma', SEN: 'sn', EGY: 'eg', GHA: 'gh',
  ALG: 'dz', COD: 'cd', TUN: 'tn', RSA: 'za',
  CIV: 'ci',
  // AFC
  JPN: 'jp', KOR: 'kr', AUS: 'au', IRN: 'ir',
  KSA: 'sa', UZB: 'uz', JOR: 'jo', IRQ: 'iq',
  QAT: 'qa',
  // OFC + other
  CPV: 'cv', NZL: 'nz',
}

const NAME_MAP: Record<string, string> = {
  'France': 'fr', 'Germany': 'de', 'Spain': 'es', 'Portugal': 'pt',
  'England': 'gb-eng', 'Netherlands': 'nl', 'Belgium': 'be', 'Croatia': 'hr',
  'Switzerland': 'ch', 'Bosnia-Herzegovina': 'ba', 'Czechia': 'cz', 'Austria': 'at',
  'Scotland': 'gb-sct', 'Turkey': 'tr', 'Norway': 'no', 'Sweden': 'se',
  'Argentina': 'ar', 'Brazil': 'br', 'Colombia': 'co', 'Ecuador': 'ec',
  'Uruguay': 'uy', 'Paraguay': 'py',
  'USA': 'us', 'Mexico': 'mx', 'Canada': 'ca', 'Panama': 'pa',
  'Haiti': 'ht', 'Curaçao': 'cw',
  'Morocco': 'ma', 'Senegal': 'sn', 'Egypt': 'eg', 'Ghana': 'gh',
  'Algeria': 'dz', 'DR Congo': 'cd', 'Tunisia': 'tn', 'South Africa': 'za',
  'Ivory Coast': 'ci',
  'Japan': 'jp', 'South Korea': 'kr', 'Australia': 'au', 'Iran': 'ir',
  'Saudi Arabia': 'sa', 'Uzbekistan': 'uz', 'Jordan': 'jo', 'Iraq': 'iq',
  'Qatar': 'qa', 'Cape Verde': 'cv', 'New Zealand': 'nz',
}

export function getFlagClass(codeOrName: string): string {
  const iso2 = TLA_MAP[codeOrName] ?? NAME_MAP[codeOrName]
  return iso2 ? `fi fi-${iso2}` : ''
}
