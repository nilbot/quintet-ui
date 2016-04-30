var renderInputMeta = function(data) {
    console.log("stats load success");
    document.getElementById("num_students").innerText = data.NumberOfStudents;
    document.getElementById("num_projects").innerText = data.NumberOfProjects;
    document.getElementById("hottest_project").innerText = data.hottestProject;
    //need to render graph here if doing it
}

var renderResult = function(data) {
    console.log("solution load success");   
     
    var table = document.getElementById('results_mapping');
    var x;
    for (x in data.assignments) {
        var m = data.assignments[x];
        var tmp = table.innerHTML;
        table.innerHTML = tmp + "<tr><td>"+m.student.Name+"</td><td>"+m.assignedProject.projectName+"</td></tr>";
    }

    document.getElementById('fitness').innerText = data.fitness;
    document.getElementById('energy').innerText = data.energyScore;
    document.getElementById('iterations').innerText = data.iterationPerformed;
    document.getElementById('strategy').innerText = data.solvingStrategy;
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
    xobj.open('GET', filename, true);
    xobj.onreadystatechange = function () {
          if (xobj.readyState == 4 && xobj.status == "200") {
            callback(xobj.responseText);
          }
    };
    xobj.send(null);  
 }
