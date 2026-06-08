export type EntityRecord = Record<string, any>;

export type Suggestion = {
  name?: string;
  source_uri?: string;
  count?: number;
  fields?: Record<string, any>;
};

export type ExtraField = {
  name: string;
  label: string;
  type: string;
  placeholder?: string;
  options?: string[];
};

export type EntityConfig = {
  formatLabel?: (entity: EntityRecord) => string;
  formatCreateData?: (name: string, suggestion?: Suggestion) => EntityRecord;
  extraFields?: ExtraField[];
};

export const comboSelectEntities: Record<string, EntityConfig> = {
  bean: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const o = e.origin || e.Origin || '';
      const r = e.roast_level || e.RoastLevel || '';
      if (o && r) return `${n} (${o} - ${r})`;
      if (o) return `${n} (${o})`;
      return n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.origin) d.origin = s.fields.origin;
        if (s.fields.roastLevel) d.roast_level = s.fields.roastLevel;
        if (s.fields.process) d.process = s.fields.process;
        if (s.fields.link) d.link = s.fields.link;
        if (s.fields.roasterName) d._source_roaster_name = s.fields.roasterName;
      }
      return d;
    },
    extraFields: [
      { name: 'origin', label: 'Origin', type: 'text', placeholder: 'e.g. Ethiopia, Colombia' },
      {
        name: 'roast_level',
        label: 'Roast Level',
        type: 'select',
        options: ['Ultra-Light', 'Light', 'Medium-Light', 'Medium', 'Medium-Dark', 'Dark']
      },
      { name: 'process', label: 'Process', type: 'text', placeholder: 'e.g. Washed, Natural, Honey' },
      { name: 'link', label: 'Link', type: 'url', placeholder: 'https://...' }
    ]
  },
  brewer: {
    formatLabel: (e) => e.name || e.Name || '',
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields?.brewerType) d.brewer_type = s.fields.brewerType;
      if (s?.fields?.link) d.link = s.fields.link;
      return d;
    },
    extraFields: [
      {
        name: 'brewer_type',
        label: 'Type',
        type: 'select',
        options: ['pourover', 'espresso', 'immersion', 'mokapot', 'coldbrew', 'cupping', 'other']
      },
      { name: 'link', label: 'Link', type: 'url', placeholder: 'https://...' }
    ]
  },
  grinder: {
    formatLabel: (e) => e.name || e.Name || '',
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.grinderType) d.grinder_type = s.fields.grinderType;
        if (s.fields.burrType) d.burr_type = s.fields.burrType;
        if (s.fields.link) d.link = s.fields.link;
      }
      return d;
    },
    extraFields: [
      { name: 'grinder_type', label: 'Type', type: 'select', options: ['Hand', 'Electric', 'Portable Electric'] },
      { name: 'burr_type', label: 'Burr Type', type: 'select', options: ['Conical', 'Flat'] },
      { name: 'link', label: 'Link', type: 'url', placeholder: 'https://...' }
    ]
  },
  recipe: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const bt = e.brewer_type || e.BrewerType || e.fields?.brewerType || '';
      return bt ? `${n} (${bt})` : n;
    },
    formatCreateData: (name) => ({ name }),
    extraFields: []
  },
  roaster: {
    formatLabel: (e) => e.name || e.Name || '',
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.location) d.location = s.fields.location;
        if (s.fields.website) d.website = s.fields.website;
      }
      return d;
    },
    extraFields: [
      { name: 'location', label: 'Location', type: 'text', placeholder: 'e.g. Portland, OR' },
      { name: 'website', label: 'Website', type: 'text', placeholder: 'https://...' }
    ]
  },
  cafe: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const l = e.location || e.Location || '';
      return l ? `${n} (${l})` : n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.location) d.location = s.fields.location;
        if (s.fields.website) d.website = s.fields.website;
      }
      return d;
    },
    extraFields: [
      { name: 'location', label: 'Location', type: 'text', placeholder: 'e.g. Portland, OR' },
      { name: 'website', label: 'Website', type: 'text', placeholder: 'https://...' }
    ]
  },
  tea: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const c = e.category || e.Category || '';
      const o = e.origin || e.Origin || '';
      if (c && o) return `${n} (${c} · ${o})`;
      if (c) return `${n} (${c})`;
      if (o) return `${n} (${o})`;
      return n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.category) d.category = s.fields.category;
        if (s.fields.subStyle) d.sub_style = s.fields.subStyle;
        if (s.fields.origin) d.origin = s.fields.origin;
        if (s.fields.cultivar) d.cultivar = s.fields.cultivar;
      }
      return d;
    },
    extraFields: [
      {
        name: 'category',
        label: 'Category',
        type: 'select',
        options: ['green', 'white', 'yellow', 'oolong', 'black', 'puerh-sheng', 'puerh-shou', 'herbal', 'blend', 'other']
      },
      { name: 'origin', label: 'Origin', type: 'text', placeholder: 'e.g. Taiwan, Yunnan' },
      { name: 'cultivar', label: 'Cultivar', type: 'text', placeholder: 'e.g. Qing Xin' }
    ]
  },
  oolongBrewer: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const st = e.style || e.Style || '';
      return st ? `${n} (${st})` : n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.style) d.style = s.fields.style;
        if (s.fields.material) d.material = s.fields.material;
        if (s.fields.link) d.link = s.fields.link;
      }
      return d;
    },
    extraFields: [
      {
        name: 'style',
        label: 'Style',
        type: 'select',
        options: ['gaiwan', 'yixing', 'kyusu', 'teapot', 'glass', 'french-press', 'tetsubin', 'other']
      },
      { name: 'material', label: 'Material', type: 'text', placeholder: 'e.g. porcelain, clay, glass' },
      { name: 'link', label: 'Link', type: 'url', placeholder: 'https://...' }
    ]
  },
  oolongVessel: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const st = e.style || e.Style || '';
      return st ? `${n} (${st})` : n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.style) d.style = s.fields.style;
        if (s.fields.material) d.material = s.fields.material;
      }
      return d;
    },
    extraFields: [
      { name: 'style', label: 'Style', type: 'select', options: ['teapot', 'mug', 'jar', 'matcha-bowl', 'other'] },
      { name: 'material', label: 'Material', type: 'text', placeholder: 'e.g. porcelain, clay, glass' }
    ]
  },
  oolongInfuser: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const st = e.style || e.Style || '';
      return st ? `${n} (${st})` : n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.style) d.style = s.fields.style;
        if (s.fields.link) d.link = s.fields.link;
      }
      return d;
    },
    extraFields: [
      { name: 'style', label: 'Style', type: 'select', options: ['basket', 'ball', 'sock', 'other'] },
      { name: 'link', label: 'Link', type: 'url', placeholder: 'https://...' }
    ]
  },
  oolongRecipe: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const st = e.style || e.Style || '';
      return st ? `${n} (${st})` : n;
    },
    formatCreateData: (name) => ({ name }),
    extraFields: []
  },
  vendor: {
    formatLabel: (e) => {
      const n = e.name || e.Name || '';
      const l = e.location || e.Location || '';
      return l ? `${n} (${l})` : n;
    },
    formatCreateData: (name, s) => {
      const d: EntityRecord = { name };
      if (s?.fields) {
        if (s.fields.location) d.location = s.fields.location;
        if (s.fields.website) d.website = s.fields.website;
      }
      return d;
    },
    extraFields: [
      { name: 'location', label: 'Location', type: 'text', placeholder: 'e.g. Taipei, Taiwan' },
      { name: 'website', label: 'Website', type: 'text', placeholder: 'https://...' }
    ]
  }
};
