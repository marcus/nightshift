/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: ['installation', 'quick-start', 'configuration'],
    },
    {
      type: 'category',
      label: 'Usage',
      items: ['tasks', 'task-reference', 'budget', 'scheduling'],
    },
    {
      type: 'category',
      label: 'Reference',
      items: ['cli-reference', 'integrations'],
    },
  ],
};

export default sidebars;
