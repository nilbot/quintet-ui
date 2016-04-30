var renderInputMeta = function(data) {
    console.log("stats load success");
    // $.each(data, function(key, value) {
    //     $('<li>', {
    //         "class": key
    //     }).text(value).appendTo($ulinc);
    // });
    console.log(data);
}

var renderResult = function(data) {
    console.log("solution load success");
    // $.each(data, function(key, value) {
    //     $('<li>', {
    //         "class": key
    //     }).text(value).appendTo($ulinc);
    // });
    console.log(data);
}

function demoLoad() {
    console.log("Loading solution and stats...")
    loadJSON(function(response) {
        var actual_JSON = JSON.parse(response);
        renderInputMeta(actual_JSON);
    }, 'testinmeta.json');
    loadJSON(function(response) {
        var actual_JSON = JSON.parse(response);
        renderResult(actual_JSON);
    }, 'testresult.json');
}


// Because Nils does not want to use jQuery at all
function loadJSON(callback, filename) {   

    var xobj = new XMLHttpRequest();
    xobj.overrideMimeType("application/json");
    xobj.open('GET', filename, true); // Replace 'my_data' with the path to your file
    xobj.onreadystatechange = function () {
          if (xobj.readyState == 4 && xobj.status == "200") {
            // Required use of an anonymous callback as .open will NOT return a value but simply returns undefined in asynchronous mode
            callback(xobj.responseText);
          }
    };
    xobj.send(null);  
 }
