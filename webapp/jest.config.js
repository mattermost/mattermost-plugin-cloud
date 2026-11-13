module.exports = {
    testEnvironment: 'jsdom',
    setupFilesAfterEnv: ['@testing-library/jest-dom/extend-expect'],
    moduleNameMapper: {
        '^@mattermost/shared/components/tooltip$': '<rootDir>/src/test_stubs/tooltip.js',
        '\\.(css|less|scss|sass)$': 'identity-obj-proxy',
    },
};