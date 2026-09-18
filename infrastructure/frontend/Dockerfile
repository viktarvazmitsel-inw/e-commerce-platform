FROM node:26.8-alpine

WORKDIR /app

RUN mkdir -p /app/node_modules && chown -R 1000:1000 /app

USER node

EXPOSE 5173

CMD ["npm", "run", "dev"]