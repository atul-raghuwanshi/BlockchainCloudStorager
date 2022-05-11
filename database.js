//3 api
// 1. store --make a watcher thread
// 2. verify
// 3. read

const express = require('express');
const path = require('path')

const app =express();

app.use(
    express.urlencoded({
      extended: true,
    })
);
  
app.use(express.json());

let commiteddb= new Map();

let uncommiteddb= new Map();

let uncommitedcount=new Map();

app.post('/upload',(req,res)=>{

    uncommiteddb.set(req.body.DocumentId,req.body.Content)
    uncommitedcount.set(req.body.DocumentId,0);

    console.log(req.body.DocumentId)
    console.log(uncommiteddb.get(req.body.DocumentId))

    res.json({"DocumentId":`${req.body.DocumentId}`,"Content":`${req.body.Content}`});
});

app.post('/verify/retrieve',(req,res)=>{

    console.log(req.body.DocumentId)
    console.log(uncommiteddb.get(req.body.DocumentId))
    res.json({"DocumentId":`${req.body.DocumentId}`,"Content":`${uncommiteddb.get(req.body.DocumentId)}`});
});

app.post('/verify/upload',(req,res)=>{
    if(uncommitedcount.has(req.body.DocumentId)==false)
    {
        uncommitedcount.set(req.body.DocumentId,1);
    }
    else{
        uncommitedcount.set(req.body.DocumentId,uncommitedcount.get(req.body.DocumentId)+1);
    }

    if(uncommitedcount.get(req.body.DocumentId)==2)
    {
        uncommitedcount.set(req.body.DocumentId,0);
        commiteddb.set(req.body.DocumentId,uncommiteddb.get(req.body.DocumentId));
    }

    console.log(req.body.DocumentId)
    console.log(uncommiteddb.get(req.body.DocumentId))
    console.log(commiteddb.get(req.body.DocumentId))
    console.log(uncommitedcount.get(req.body.DocumentId));
    res.json({"DocumentId":`${req.body.DocumentId}`,"Content":""});
});

app.post('/read',(req,res)=>{

    console.log(req.body.DocumentId)
    console.log(commiteddb.get(req.body.DocumentId))

    res.json({"DocumentId":`${req.body.DocumentId}`,"Content":`${commiteddb.get(req.body.DocumentId)}`});
});

const PORT = process.env.PORT || 5000;

app.listen(PORT, ()=> console.log(`Server started on port ${PORT}`));

